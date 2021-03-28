#include "pybind11/pybind11.h"
#include "pybind11/stl.h"

#include "client/profile/profile.hpp"

#include "tenncor/tenncor.hpp"

namespace py = pybind11;

PYBIND11_MODULE(profile, m)
{
	m.doc() = "profile teq graphs";

	m
		// ==== to stdout functions ====
        .def("remote_profile", dbg::profile::remote_profile,
        "Profile graph of tensors and report to remote address",
        py::arg("addr"),
        py::arg("roots"));
}
