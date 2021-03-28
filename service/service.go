package service

import (
	"fmt"
	"sync"

	pb "github.com/mingkaic/onnx_go/onnx"
	log "github.com/sirupsen/logrus"

	"github.com/mingkaic/accretion/data"
)

type (
	GraphService interface {
		CreateGraphProfile(*pb.ModelProto, map[string]uint64) error
	}

	graphService struct{}
)

const batchsize = 5

func NewGraphService() GraphService {
	return &graphService{}
}

func (graphService) CreateGraphProfile(
	model *pb.ModelProto, runtimes map[string]uint64) error {
	pbGraph := model.GetGraph()
	graph, annotations, err := transformGraph(pbGraph)
	if err != nil {
		return err
	}
	annotationList := make([]*data.Annotation, 0, len(annotations))
	for _, annotation := range annotations {
		annotationList = append(annotationList, annotation)
	}
	nodes := make([]interface{}, 0, len(graph))
	for id, node := range graph {
		if runtime, ok := runtimes[id]; ok {
			if fnc, ok := node.(*data.Func); ok {
				fnc.Runtime = runtime
			}
		}
		nodes = append(nodes, node)
	}
	return data.WithTx(func(tx data.Txn) (err error) {
		//log.Debug("saving annotations")
		//if err = data.CreateNode(tx, annotationList); err != nil {
		//return
		//}
		nnodes := len(nodes)
		nbatches := nnodes / batchsize
		log.Debugf("saving graph nodes %d", nnodes)
		log.Debugf("saving by %d batches", nbatches)
		var (
			wg      sync.WaitGroup
			errChan = make(chan error, 0)
		)
		for i := 0; i < nbatches; i++ {
			startIdx := i * batchsize
			data.AsyncCreateNode(&wg, errChan, fmt.Sprintf("%d", i), tx, nodes[startIdx:startIdx+batchsize])
		}
		if nbatches*batchsize < nnodes {
			data.AsyncCreateNode(&wg, errChan, fmt.Sprintf("%d", nbatches+1), tx, nodes[nbatches*batchsize:])
		}
		go func() {
			for err = range errChan {
				log.Error(err)
			}
		}()
		wg.Wait()
		return
	})
}

func transformGraph(graph *pb.GraphProto) (map[string]interface{}, map[string]*data.Annotation, error) {
	var (
		err  error
		leaf *data.Leaf
		fnc  *data.Func

		inputs = graph.GetInput()
		inits  = graph.GetInitializer()
		sinits = graph.GetSparseInitializer()
		funcs  = graph.GetNode()

		nodes                        = make(map[string]interface{})
		annotationEdges, annotations = getAnnotations(graph.GetQuantizationAnnotation())
	)
	for _, input := range inputs {
		id := input.GetName()
		leaf = transformPlaceholder(input)
		leaf.Annotations = annotationEdges[id]
		nodes[id] = leaf
	}
	for _, init := range inits {
		id := init.GetName()
		if leaf, err = transformVariable(init); err != nil {
			return nil, nil, err
		}
		leaf.Annotations = annotationEdges[id]
		nodes[id] = leaf
	}
	for _, sinit := range sinits {
		id := sinit.GetValues().GetName()
		if leaf, err = transformSVariable(sinit); err != nil {
			return nil, nil, err
		}
		leaf.Annotations = annotationEdges[id]
		nodes[id] = leaf
	}
	for _, pbfnc := range funcs {
		if subgraph, ok := getSubgraph(pbfnc); ok {
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
			id := pbfnc.GetName()
			if fnc, err = transformFunc(pbfnc, nodes, annotations); err != nil {
				return nil, nil, err
			}
			nodes[id] = fnc
		}
	}
	return nodes, annotations, nil
}

func getAnnotations(qAnnotations []*pb.TensorAnnotation) (map[string][]string, map[string]*data.Annotation) {
	var (
		edge        []string
		edges       = make(map[string][]string)
		annotations = make(map[string]*data.Annotation)
	)
	for _, qAnnotation := range qAnnotations {
		id := qAnnotation.GetTensorName()
		entries := qAnnotation.GetQuantParameterTensorNames()
		edge = make([]string, len(entries))
		edges[id] = edge
		for i, entry := range entries {
			annotation := data.NewAnnotation(entry.GetKey(), entry.GetValue())
			annotations[annotation.Id] = annotation
			edge[i] = annotation.Id
		}
	}
	return edges, annotations
}

func transformPlaceholder(input *pb.ValueInfoProto) *data.Leaf {
	return &data.Leaf{
		Id: input.GetName(),
	}
}

func transformVariable(init *pb.TensorProto) (*data.Leaf, error) {
	var (
		tensordata []float64
		dtype      = pb.TensorProto_DataType(init.GetDataType())
	)
	switch dtype {
	case pb.TensorProto_DOUBLE:
		tensordata = init.GetDoubleData()
	case pb.TensorProto_FLOAT:
		fdata := init.GetFloatData()
		tensordata = make([]float64, 0, len(fdata))
		for _, f := range fdata {
			tensordata = append(tensordata, float64(f))
		}
	case pb.TensorProto_INT32, pb.TensorProto_UINT8, pb.TensorProto_UINT16, pb.TensorProto_INT16:
		idata := init.GetInt32Data()
		tensordata = make([]float64, 0, len(idata))
		for _, i := range idata {
			tensordata = append(tensordata, float64(i))
		}
	case pb.TensorProto_UINT32, pb.TensorProto_UINT64:
		udata := init.GetUint64Data()
		tensordata = make([]float64, 0, len(udata))
		for _, u := range udata {
			tensordata = append(tensordata, float64(u))
		}
	case pb.TensorProto_INT64:
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
	return &data.Leaf{
		Id:    init.GetName(),
		Data:  tensordata,
		Shape: dims,
	}, nil
}

func transformSVariable(init *pb.SparseTensorProto) (*data.Leaf, error) {
	leaf, err := transformVariable(init.GetValues())
	if err != nil {
		return nil, err
	}
	inners := init.GetIndices().GetInt32Data()
	outers := init.GetDims()
	leaf.Sinfo = &data.SparseInfo{
		NonZeros:     len(inners),
		Indices:      inners,
		OuterIndices: outers,
	}
	return leaf, nil
}

func transformFunc(fnc *pb.NodeProto, nodes map[string]interface{}, annotations map[string]*data.Annotation) (*data.Func, error) {
	var (
		val   interface{}
		atype pb.AttributeProto_AttributeType

		id     = fnc.GetName()
		attrs  = fnc.GetAttribute()
		opname = fnc.GetOpType()

		annotationEdges = make([]string, len(attrs))
	)
	for i, attr := range attrs {
		atype = attr.GetType()
		switch atype {
		case pb.AttributeProto_FLOAT:
			val = attr.GetF()
		case pb.AttributeProto_INT:
			val = attr.GetI()
		case pb.AttributeProto_STRING:
			val = attr.GetS()
		case pb.AttributeProto_FLOATS:
			val = attr.GetFloats()
		case pb.AttributeProto_INTS:
			val = attr.GetInts()
		case pb.AttributeProto_STRINGS:
			val = attr.GetStrings()
		case pb.AttributeProto_TENSOR:
			val = nodes[attr.GetT().GetName()]
		case pb.AttributeProto_SPARSE_TENSOR:
			val = nodes[attr.GetSparseTensor().GetValues().GetName()]
		case pb.AttributeProto_TENSORS:
			pbtens := attr.GetTensors()
			tens := make([]interface{}, len(pbtens))
			for j, tensor := range pbtens {
				tens[j] = nodes[tensor.GetName()]
			}
			val = tens
		case pb.AttributeProto_SPARSE_TENSORS:
			pbtens := attr.GetSparseTensors()
			tens := make([]interface{}, len(pbtens))
			for j, tensor := range pbtens {
				tens[j] = nodes[tensor.GetValues().GetName()]
			}
			val = tens
		default:
			return nil, fmt.Errorf("unsupported attribute type %s", atype)
		}
		annotation := data.NewAnnotation(attr.GetName(), fmt.Sprint(val))
		annotations[annotation.Id] = annotation
		annotationEdges[i] = annotation.Id
	}
	inputs := fnc.GetInput()
	args := make([]string, len(inputs))
	for i, input := range inputs {
		args[i] = fmt.Sprint(nodes[input])
	}
	return &data.Func{
		Id:          id,
		Opname:      opname,
		Args:        args,
		Annotations: annotationEdges,
	}, nil
}

func getSubgraph(fnc *pb.NodeProto) (*pb.GraphProto, bool) {
	attrs := fnc.GetAttribute()
	for _, attr := range attrs {
		if attr.GetType() == pb.AttributeProto_GRAPH {
			return attr.GetG(), true
		}
	}
	return nil, false
}
