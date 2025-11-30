package Resource_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/official-biswadeb941/Infermal_v2/Modules/Resource"
)

// helper to clear environment flags
func clearEnv() {
	_ = os.Unsetenv("HAS_GPU")
	_ = os.Unsetenv("HAS_TPU")
}

func TestDetectDevices_CPUOnly(t *testing.T) {
	clearEnv()

	info := Resource.DetectDevices()

	assert.NotEmpty(t, info.Devices, "CPU device should be detected")
	assert.Equal(t, Resource.CPU, info.Devices[0].Type)
	assert.True(t, info.Devices[0].Cores > 0)

	assert.Contains(t, info.Warnings, "No GPU detected")
	assert.Contains(t, info.Warnings, "No TPU detected")

	assert.Empty(t, info.Errors)
}

func TestDetectDevices_WithGPU(t *testing.T) {
	clearEnv()
	_ = os.Setenv("HAS_GPU", "1")

	info := Resource.DetectDevices()

	foundGPU := false
	for _, d := range info.Devices {
		if d.Type == Resource.GPU {
			foundGPU = true
			assert.Equal(t, "Generic GPU", d.Name)
		}
	}

	assert.True(t, foundGPU, "GPU should be detected")
	assert.Contains(t, info.Info, "Generic GPU detected")
}

func TestDetectDevices_WithTPU(t *testing.T) {
	clearEnv()
	_ = os.Setenv("HAS_TPU", "1")

	info := Resource.DetectDevices()

	foundTPU := false
	for _, d := range info.Devices {
		if d.Type == Resource.TPU {
			foundTPU = true
			assert.Equal(t, "Generic TPU", d.Name)
		}
	}

	assert.True(t, foundTPU, "TPU should be detected")
	assert.Contains(t, info.Info, "Generic TPU detected")
}

func TestDetectDevices_GPUAndTPU(t *testing.T) {
	clearEnv()
	_ = os.Setenv("HAS_GPU", "1")
	_ = os.Setenv("HAS_TPU", "1")

	info := Resource.DetectDevices()

	var gpu, tpu bool
	for _, d := range info.Devices {
		if d.Type == Resource.GPU {
			gpu = true
		}
		if d.Type == Resource.TPU {
			tpu = true
		}
	}

	assert.True(t, gpu)
	assert.True(t, tpu)
	assert.Empty(t, info.Warnings)
}

func TestGetBestDevice(t *testing.T) {
	clearEnv()
	_ = os.Setenv("HAS_GPU", "1")
	_ = os.Setenv("HAS_TPU", "1")

	best, info, warn, errs := Resource.GetBestDevice()

	assert.NotNil(t, best)
	assert.Empty(t, errs)
	assert.NotEmpty(t, info)
	assert.Empty(t, warn)

	// TPU has highest score = 80
	assert.Equal(t, Resource.TPU, best.Type)
	assert.Equal(t, 80, best.Score)
}

func TestGetBestDevice_NoDevices(t *testing.T) {
	// Simulate zero CPU core scenario? Not possible directly.
	// Instead: unset GPU and TPU and rely on CPU being always present.
	// So this test ensures stability rather than forcing errors.

	clearEnv()
	best, info, warn, errs := Resource.GetBestDevice()

	assert.NotNil(t, info)
	assert.NotNil(t, warn)
	assert.Empty(t, errs, "CPU should always exist on Go runtime")
	assert.Equal(t, Resource.CPU, best.Type)
}

func TestDeviceString_CPU(t *testing.T) {
	d := Resource.Device{
		Type:  Resource.CPU,
		ID:    0,
		Name:  "CPU x8 cores",
		Score: 10,
		Cores: 8,
	}

	out := d.String()
	assert.Contains(t, out, "cpu:0")
	assert.Contains(t, out, "CPU x8 cores")
	assert.Contains(t, out, "score=10")
}

func TestDeviceString_Generic(t *testing.T) {
	d := Resource.Device{
		Type:  Resource.GPU,
		ID:    2,
		Name:  "Test GPU",
		Score: 40,
	}

	out := d.String()
	assert.Contains(t, out, "gpu:2")
	assert.Contains(t, out, "Test GPU")
	assert.Contains(t, out, "score=40")
}
