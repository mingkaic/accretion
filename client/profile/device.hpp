#include "internal/eigen/eigen.hpp"

#ifndef DBG_PROFILE_DEVICE_HPP
#define DBG_PROFILE_DEVICE_HPP

namespace dbg
{

namespace profile
{

#define TIME(action)\
std::chrono::high_resolution_clock::time_point start =\
	std::chrono::high_resolution_clock::now();\
action;\
size_t stat = std::chrono::duration_cast<std::chrono::nanoseconds>(\
	std::chrono::high_resolution_clock::now() - start).count();

struct ProfilerDevice final : public teq::iDevice
{
	ProfilerDevice (teq::iDevice& dev) : dev_(&dev) {}

	void calc (teq::iTensor& tens, size_t cache_ttl) override
	{
		TIME(dev_->calc(tens, cache_ttl + 1));
		stats_.emplace(&tens, stat);
	}

	teq::TensMapT<size_t> stats_;

private:
	teq::iDevice* dev_;
};

}

}

#endif // DBG_PROFILE_DEVICE_HPP
