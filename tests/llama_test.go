package tests

import (
	"testing"

	"github.com/simonxluo/Shepherd/internal/comm/utils"
)

func TestParseLlamacppDeviceList(t *testing.T) {
	// Test 1: No GPU
	realOutput := "Available devices:\n  (none)\n"
	devices := utils.ParseLlamacppDeviceList(realOutput)
	if len(devices) != 0 {
		t.Errorf("Test 1 (no GPU): got %d devices, want 0: %v", len(devices), devices)
	}

	// Test 2: CUDA + ROCm with duplicates (dedup)
	simulatedOutput := "ggml_cuda_init: GGML_CUDA_FORCE_MMQ:    no\nggml_cuda_init: GGML_CUDA_FORCE_CUBLAS: no\nggml_cuda_init: found 1 CUDA devices:\nAvailable devices:\n  CUDA0: NVIDIA GeForce RTX 4090 (24564 MiB, 23424 MiB free)\n  CUDA0: NVIDIA GeForce RTX 4090 (24564 MiB, 23424 MiB free)\n  ROCm0: AMD Radeon RX 7900 XTX (24560 MiB, 22000 MiB free)\n"
	devices = utils.ParseLlamacppDeviceList(simulatedOutput)
	if len(devices) != 2 {
		t.Errorf("Test 2 (dedup): got %d devices, want 2: %v", len(devices), devices)
	}

	// Test 3: Log noise before "Available devices:" should not pollute
	logOutput := "INFO: loading model\nWARNING: something\nAvailable devices:\n  CUDA0: NVIDIA GeForce RTX 3090 (24576 MiB, 20321 MiB free)\n  Metal0: Apple M2 Max (32768 MiB, 28000 MiB free)\n\nDone.\n"
	devices = utils.ParseLlamacppDeviceList(logOutput)
	if len(devices) != 2 {
		t.Errorf("Test 3 (log noise): got %d devices, want 2: %v", len(devices), devices)
	}

	// Test 4: Invalid prefixes (INFO, http) should be rejected
	badOutput := "Available devices:\n  INFO: not a device\n  http: also not a device\n  CUDA0: Real Device (1024 MiB, 512 MiB free)\n"
	devices = utils.ParseLlamacppDeviceList(badOutput)
	if len(devices) != 1 {
		t.Errorf("Test 4 (invalid prefixes): got %d devices, want 1: %v", len(devices), devices)
	}
	if len(devices) == 1 && devices[0] != "CUDA0: Real Device (1024 MiB, 512 MiB free)" {
		t.Errorf("Test 4: got %q, want CUDA0 line", devices[0])
	}

	// Test 5: All known device types
	allTypes := "Available devices:\n  CUDA0: Device A (1024 MiB, 512 MiB free)\n  ROCm0: Device B (2048 MiB, 1024 MiB free)\n  Vulkan0: Device C (4096 MiB, 2048 MiB free)\n  Metal0: Device D (8192 MiB, 4096 MiB free)\n  SYCL0: Device E (16384 MiB, 8192 MiB free)\n  CPU0: CPU Device (32768 MiB, 16384 MiB free)\n"
	devices = utils.ParseLlamacppDeviceList(allTypes)
	if len(devices) != 6 {
		t.Errorf("Test 5 (all types): got %d devices, want 6: %v", len(devices), devices)
	}
}
