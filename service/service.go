package service

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"math"
	"sync"

	"github.com/mingkaic/accretion/proto/profile"
	"github.com/mingkaic/onnx_go/onnx"
	log "github.com/sirupsen/logrus"

	"github.com/mingkaic/accretion/data"
	"github.com/mingkaic/accretion/proto/storage"
)

type (
	GraphService interface {
		ListGraphProfiles() ([]string, error)
		GetGraphProfile(string) ([]*profile.SigmaNode, []*profile.SigmaEdge, error)
		CreateGraphProfile(string, *onnx.ModelProto, map[string]*profile.FuncInfo) error
	}

	graphService struct{}

	ProfileGroupbyEntry struct {
		ProfileId string `json:"profile_id"`
		Count     int    `json:"count"`
	}

	ProfileGroupby struct {
		GroupBy []*ProfileGroupbyEntry `json:"@groupby"`
	}

	ProfileNode struct {
		Id, Label string
		Arg       []struct {
			Id string
		}
	}
)

const (
	batchsize     = 8
	profileLookup = `{
	profiles(func:has(profile_id)) @groupby(profile_id) {
		count(uid)
	}
}`
	nodesLookupFmt = `{
	nodes(func: allofterms(profile_id, "%s")) {
		id
		label
		arg {
			id
		}
	}
}`
)

func NewGraphService() GraphService {
	return &graphService{}
}

func (graphService) ListGraphProfiles() ([]string, error) {
	var profiles []string
	if err := data.WithTx(func(tx *data.Txn) (err error) {
		var (
			b        []byte
			response = make(map[string][]*ProfileGroupby)
		)
		b, err = data.QueryNode(tx, profileLookup)
		if err != nil {
			return
		}
		if err = json.Unmarshal(b, &response); err != nil {
			return
		}
		entries, ok := response["profiles"]
		if !ok || len(entries) < 1 {
			err = fmt.Errorf("invalid response from dgraph: %s", string(b))
			return
		}
		profiles = make([]string, len(entries[0].GroupBy))
		for i, profile := range entries[0].GroupBy {
			profiles[i] = profile.ProfileId
		}
		return
	}); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (graphService) GetGraphProfile(id string) ([]*profile.SigmaNode, []*profile.SigmaEdge, error) {
	var (
		nodes []*profile.SigmaNode
		edges []*profile.SigmaEdge
	)
	if err := data.WithTx(func(tx *data.Txn) (err error) {
		var (
			b        []byte
			response = make(map[string][]*ProfileNode)
		)
		b, err = data.QueryNode(tx, fmt.Sprintf(nodesLookupFmt, id))
		if err != nil {
			return
		}
		if err = json.Unmarshal(b, &response); err != nil {
			return
		}
		profNodes, ok := response["nodes"]
		if !ok {
			err = fmt.Errorf("invalid response from dgraph: %s", string(b))
			return
		}
		nodes = make([]*profile.SigmaNode, len(profNodes))
		row := int(math.Sqrt(float64(len(nodes))))
		for i, profNode := range profNodes {
			nodes[i] = &profile.SigmaNode{
				Id:    profNode.Id,
				Label: profNode.Label,
				X:     int64(i % row),
				Y:     int64(i / row),
				Size:  5,
			}
			if len(profNode.Arg) > 0 {
				for _, arg := range profNode.Arg {
					id := uuid.NewString()
					edges = append(edges, &profile.SigmaEdge{
						Id:     id,
						Source: profNode.Id,
						Target: arg.Id,
					})
				}
			}
		}
		return
	}); err != nil {
		return nil, nil, err
	}
	return nodes, edges, nil
}

func (graphService) CreateGraphProfile(profileId string,
	model *onnx.ModelProto, opData map[string]*profile.FuncInfo) error {
	pbGraph := model.GetGraph()
	graph, annotations, err := transformGraph(pbGraph)
	if err != nil {
		return err
	}
	annotationList := make([]interface{}, 0, len(annotations))
	for _, annotation := range annotations {
		annotationList = append(annotationList, annotation)
	}
	for id, node := range graph {
		node.ProfileId = profileId
		node.Args = make([]*data.TenncorNode, len(node.ArgIds))
		for i, argId := range node.ArgIds {
			node.Args[i] = graph[argId]
		}
		if op, ok := opData[id]; ok {
			node.Runtime = op.GetRuntime()
			if denseData := op.GetDenseData(); denseData != nil {
				if variable, err := transformVariable(denseData); err != nil {
					node.Shape = variable.Shape
					node.Data = variable.Data
				}
			} else if sparseData := op.GetSparseData(); sparseData != nil {
				if variable, err := transformSVariable(sparseData); err != nil {
					node.Shape = variable.Shape
					node.Data = variable.Data
					node.Sinfo = variable.Sinfo
				}
			}
		}
	}
	outputs := pbGraph.GetOutput()
	roots := make([]interface{}, len(outputs))
	for i, output := range outputs {
		roots[i] = graph[output.GetName()]
	}
	return data.WithTx(func(tx *data.Txn) (err error) {
		var (
			wg      sync.WaitGroup
			blobWg  sync.WaitGroup
			errChan = make(chan error, 0)
		)
		go func() {
			for err = range errChan {
				log.Error(err)
			}
		}()
		// saving blob
		log.Debug("saving node blob")
		for id, node := range graph {
			blob := &storage.BlobStorage{
				Data: node.Data,
			}
			if node.Sinfo != nil {
				blob.Indices = node.Sinfo.Indices
				blob.OuterIndices = node.Sinfo.OuterIndices
			}
			data.AsyncSaveBlob(&blobWg, errChan, profileId, id, blob)
		}
		log.Debug("saving roots")
		data.BatchCreateNodes(&wg, errChan, tx, roots, batchsize)
		wg.Wait()
		blobWg.Wait()
		return
	})
}

func transformGraph(graph *onnx.GraphProto) (map[string]*data.TenncorNode, map[string]*data.Annotation, error) {
	var (
		err  error
		node *data.TenncorNode

		inputs = graph.GetInput()
		inits  = graph.GetInitializer()
		sinits = graph.GetSparseInitializer()
		funcs  = graph.GetNode()

		nodes                        = make(map[string]*data.TenncorNode)
		annotationEdges, annotations = getAnnotations(graph.GetQuantizationAnnotation())
	)
	for _, input := range inputs {
		id := input.GetName()
		node = transformPlaceholder(input)
		node.Annotations = annotationEdges[id]
		nodes[id] = node
	}
	for _, init := range inits {
		id := init.GetName()
		if node, err = transformVariable(init); err != nil {
			return nil, nil, err
		}
		node.Annotations = annotationEdges[id]
		nodes[id] = node
	}
	for _, sinit := range sinits {
		id := sinit.GetValues().GetName()
		if node, err = transformSVariable(sinit); err != nil {
			return nil, nil, err
		}
		node.Annotations = annotationEdges[id]
		nodes[id] = node
	}
	for _, pbFnc := range funcs {
		if subgraph, ok := getSubgraph(pbFnc); ok {
			subNodes, subAnnotations, err := transformGraph(subgraph)
			if err != nil {
				return nil, nil, err
			}
			for k, v := range subNodes {
				nodes[k] = v
			}
			for k, v := range subAnnotations {
				annotations[k] = v
			}
		} else {
			id := pbFnc.GetName()
			if node, err = transformFunc(pbFnc, nodes, annotations); err != nil {
				return nil, nil, err
			}
			nodes[id] = node
		}
	}
	return nodes, annotations, nil
}

func getAnnotations(qAnnotations []*onnx.TensorAnnotation) (map[string][]*data.Annotation, map[string]*data.Annotation) {
	var (
		edge        []*data.Annotation
		edges       = make(map[string][]*data.Annotation)
		annotations = make(map[string]*data.Annotation)
	)
	for _, qAnnotation := range qAnnotations {
		id := qAnnotation.GetTensorName()
		entries := qAnnotation.GetQuantParameterTensorNames()
		edge = make([]*data.Annotation, len(entries))
		edges[id] = edge
		for i, entry := range entries {
			annotation := data.NewAnnotation(entry.GetKey(), entry.GetValue())
			annotations[annotation.Id] = annotation
			edge[i] = annotation
		}
	}
	return edges, annotations
}

func transformPlaceholder(input *onnx.ValueInfoProto) *data.TenncorNode {
	id := input.GetName()
	return &data.TenncorNode{
		Uid: fmt.Sprintf("_:%s", id),
		Id:  id,
	}
}

func transformVariable(init *onnx.TensorProto) (*data.TenncorNode, error) {
	var (
		tensordata []float64
		dtype      = onnx.TensorProto_DataType(init.GetDataType())
	)
	switch dtype {
	case onnx.TensorProto_DOUBLE:
		tensordata = init.GetDoubleData()
	case onnx.TensorProto_FLOAT:
		fdata := init.GetFloatData()
		tensordata = make([]float64, 0, len(fdata))
		for _, f := range fdata {
			tensordata = append(tensordata, float64(f))
		}
	case onnx.TensorProto_INT32, onnx.TensorProto_UINT8, onnx.TensorProto_UINT16, onnx.TensorProto_INT16:
		idata := init.GetInt32Data()
		tensordata = make([]float64, 0, len(idata))
		for _, i := range idata {
			tensordata = append(tensordata, float64(i))
		}
	case onnx.TensorProto_UINT32, onnx.TensorProto_UINT64:
		udata := init.GetUint64Data()
		tensordata = make([]float64, 0, len(udata))
		for _, u := range udata {
			tensordata = append(tensordata, float64(u))
		}
	case onnx.TensorProto_INT64:
		idata := init.GetInt64Data()
		tensordata = make([]float64, 0, len(idata))
		for _, i := range idata {
			tensordata = append(tensordata, float64(i))
		}
	default:
		return nil, fmt.Errorf("bad variable type %s", dtype)
	}

	ds := init.GetDims()
	dims := make([]uint64, len(ds))
	for i, d := range ds {
		dims[i] = uint64(d)
	}
	id := init.GetName()
	return &data.TenncorNode{
		Uid:   fmt.Sprintf("_:%s", id),
		Id:    id,
		Data:  tensordata,
		Shape: dims,
	}, nil
}

func transformSVariable(init *onnx.SparseTensorProto) (*data.TenncorNode, error) {
	leaf, err := transformVariable(init.GetValues())
	if err != nil {
		return nil, err
	}
	inners := init.GetIndices().GetInt32Data()
	outers := init.GetDims()
	leaf.Sinfo = &data.SparseInfo{
		Indices:      inners,
		OuterIndices: outers,
	}
	return leaf, nil
}

func transformFunc(fnc *onnx.NodeProto, nodes map[string]*data.TenncorNode, annotations map[string]*data.Annotation) (*data.TenncorNode, error) {
	var (
		val   interface{}
		atype onnx.AttributeProto_AttributeType

		id     = fnc.GetName()
		attrs  = fnc.GetAttribute()
		opname = fnc.GetOpType()

		annotationEdges = make([]*data.Annotation, len(attrs))
	)
	for i, attr := range attrs {
		atype = attr.GetType()
		switch atype {
		case onnx.AttributeProto_FLOAT:
			val = attr.GetF()
		case onnx.AttributeProto_INT:
			val = attr.GetI()
		case onnx.AttributeProto_STRING:
			val = attr.GetS()
		case onnx.AttributeProto_FLOATS:
			val = attr.GetFloats()
		case onnx.AttributeProto_INTS:
			val = attr.GetInts()
		case onnx.AttributeProto_STRINGS:
			val = attr.GetStrings()
		case onnx.AttributeProto_TENSOR:
			val = nodes[attr.GetT().GetName()]
		case onnx.AttributeProto_SPARSE_TENSOR:
			val = nodes[attr.GetSparseTensor().GetValues().GetName()]
		case onnx.AttributeProto_TENSORS:
			pbTens := attr.GetTensors()
			tens := make([]interface{}, len(pbTens))
			for j, tensor := range pbTens {
				tens[j] = nodes[tensor.GetName()]
			}
			val = tens
		case onnx.AttributeProto_SPARSE_TENSORS:
			pbTens := attr.GetSparseTensors()
			tens := make([]interface{}, len(pbTens))
			for j, tensor := range pbTens {
				tens[j] = nodes[tensor.GetValues().GetName()]
			}
			val = tens
		default:
			return nil, fmt.Errorf("unsupported attribute type %s", atype)
		}
		annotation := data.NewAnnotation(attr.GetName(), fmt.Sprint(val))
		annotations[annotation.Id] = annotation
		annotationEdges[i] = annotation
	}
	inputs := fnc.GetInput()
	argIds := make([]string, len(inputs))
	for i, input := range inputs {
		argIds[i] = input
	}
	return &data.TenncorNode{
		Uid:         fmt.Sprintf("_:%s", id),
		Id:          id,
		Label:       opname,
		Annotations: annotationEdges,
		ArgIds:      argIds,
	}, nil
}

func getSubgraph(fnc *onnx.NodeProto) (*onnx.GraphProto, bool) {
	attrs := fnc.GetAttribute()
	for _, attr := range attrs {
		if attr.GetType() == onnx.AttributeProto_GRAPH {
			return attr.GetG(), true
		}
	}
	return nil, false
}
