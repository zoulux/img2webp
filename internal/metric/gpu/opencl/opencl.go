//go:build linux || windows || darwin

package opencl

/*
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -lOpenCL
#cgo windows CFLAGS: -I"C:/Program Files/NVIDIA GPU Computing Toolkit/CUDA/include"
#cgo windows LDFLAGS: -L"C:/Program Files/NVIDIA GPU Computing Toolkit/CUDA/lib/x64" -lOpenCL
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework OpenCL

#ifdef __APPLE__
#include <OpenCL/opencl.h>
#else
#include <CL/cl.h>
#endif

// OpenCL kernel source code
static const char* kernelSource = R"(
__kernel void ssim_kernel(
    __global const uchar* imgA,
    __global const uchar* imgB,
    __global float* totalDiff,
    const uint width,
    const uint height
) {
    uint x = get_global_id(0);
    uint y = get_global_id(1);
    if (x >= width || y >= height) return;

    uint idx = (y * width + x) * 4;
    float diff = fabs((float)(imgA[idx] - imgB[idx]))
               + fabs((float)(imgA[idx+1] - imgB[idx+1]))
               + fabs((float)(imgA[idx+2] - imgB[idx+2]));

    atomic_add(totalDiff, diff);
}

__kernel void edge_kernel(
    __global const uchar* imgA,
    __global const uchar* imgB,
    __global float* totalDiff,
    const uint width,
    const uint height
) {
    uint x = get_global_id(0);
    uint y = get_global_id(1);
    if (x >= width - 1 || y >= height - 1) return;

    uint idx1 = (y * width + x) * 4;
    uint idx2 = (y * width + x + 1) * 4;

    float edgeA = fabs((float)(imgA[idx1] - imgA[idx2]))
                + fabs((float)(imgA[idx1+1] - imgA[idx2+1]))
                + fabs((float)(imgA[idx1+2] - imgA[idx2+2]));

    float edgeB = fabs((float)(imgB[idx1] - imgB[idx2]))
                + fabs((float)(imgB[idx1+1] - imgB[idx2+1]))
                + fabs((float)(imgB[idx1+2] - imgB[idx2+2]));

    atomic_add(totalDiff, fabs(edgeA - edgeB));
}
)";

// OpenCL resources
static cl_platform_id platform = NULL;
static cl_device_id device = NULL;
static cl_context context = NULL;
static cl_command_queue queue = NULL;
static cl_program program = NULL;
static cl_kernel ssim_kernel = NULL;
static cl_kernel edge_kernel = NULL;

// Select best device: discrete GPU > integrated GPU > CPU
static cl_device_id selectBestDevice(cl_platform_id plat) {
    cl_int err;
    cl_device_id device = NULL;
    cl_uint numDevices;

    // 1. Try GPU devices
    err = clGetDeviceIDs(plat, CL_DEVICE_TYPE_GPU, 0, NULL, &numDevices);
    if (err == CL_SUCCESS && numDevices > 0) {
        cl_device_id* devices = (cl_device_id*)malloc(sizeof(cl_device_id) * numDevices);
        if (devices == NULL) return NULL;

        err = clGetDeviceIDs(plat, CL_DEVICE_TYPE_GPU, numDevices, devices, NULL);
        if (err != CL_SUCCESS) {
            free(devices);
            return NULL;
        }

        // Prefer discrete GPU (non-unified memory)
        for (cl_uint i = 0; i < numDevices; i++) {
            cl_bool unifiedMemory = CL_TRUE;
            clGetDeviceInfo(devices[i], CL_DEVICE_HOST_UNIFIED_MEMORY,
                           sizeof(cl_bool), &unifiedMemory, NULL);

            if (unifiedMemory == CL_FALSE) {
                device = devices[i];
                break;
            }
        }

        // No discrete GPU, use first available GPU (integrated)
        if (device == NULL && numDevices > 0) {
            device = devices[0];
        }

        free(devices);
        if (device != NULL) return device;
    }

    // 2. Fallback to CPU
    err = clGetDeviceIDs(plat, CL_DEVICE_TYPE_CPU, 1, &device, NULL);
    if (err == CL_SUCCESS) {
        return device;
    }

    return NULL;
}

static int initOpenCL() {
    cl_int err;

    // Get platform
    cl_uint numPlatforms;
    err = clGetPlatformIDs(1, &platform, &numPlatforms);
    if (err != CL_SUCCESS || numPlatforms == 0) return 0;

    // Select best device
    device = selectBestDevice(platform);
    if (device == NULL) return 0;

    // Create context
    context = clCreateContext(NULL, 1, &device, NULL, NULL, &err);
    if (err != CL_SUCCESS || context == NULL) return 0;

    // Create command queue
    queue = clCreateCommandQueue(context, device, 0, &err);
    if (err != CL_SUCCESS || queue == NULL) {
        clReleaseContext(context);
        context = NULL;
        return 0;
    }

    // Create and build program
    size_t srcLen = strlen(kernelSource);
    program = clCreateProgramWithSource(context, 1, &kernelSource, &srcLen, &err);
    if (err != CL_SUCCESS || program == NULL) {
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
        queue = NULL;
        context = NULL;
        return 0;
    }

    err = clBuildProgram(program, 1, &device, NULL, NULL, NULL);
    if (err != CL_SUCCESS) {
        clReleaseProgram(program);
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
        program = NULL;
        queue = NULL;
        context = NULL;
        return 0;
    }

    // Create kernels
    ssim_kernel = clCreateKernel(program, "ssim_kernel", &err);
    if (err != CL_SUCCESS || ssim_kernel == NULL) {
        clReleaseProgram(program);
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
        program = NULL;
        queue = NULL;
        context = NULL;
        return 0;
    }

    edge_kernel = clCreateKernel(program, "edge_kernel", &err);
    if (err != CL_SUCCESS || edge_kernel == NULL) {
        clReleaseKernel(ssim_kernel);
        clReleaseProgram(program);
        clReleaseCommandQueue(queue);
        clReleaseContext(context);
        ssim_kernel = NULL;
        program = NULL;
        queue = NULL;
        context = NULL;
        return 0;
    }

    return 1;
}

static int isOpenCLAvailable() {
    return (device != NULL && context != NULL && queue != NULL &&
            ssim_kernel != NULL && edge_kernel != NULL) ? 1 : 0;
}

static void cleanupOpenCL() {
    if (ssim_kernel) { clReleaseKernel(ssim_kernel); ssim_kernel = NULL; }
    if (edge_kernel) { clReleaseKernel(edge_kernel); edge_kernel = NULL; }
    if (program) { clReleaseProgram(program); program = NULL; }
    if (queue) { clReleaseCommandQueue(queue); queue = NULL; }
    if (context) { clReleaseContext(context); context = NULL; }
}

static float computeSSIM(int width, int height, const uint8_t* dataA, const uint8_t* dataB) {
    cl_int err;
    cl_mem bufferA = NULL, bufferB = NULL, bufferResult = NULL;
    float result = 0.0f;

    size_t dataSize = (size_t)width * (size_t)height * 4;

    // Create buffers
    bufferA = clCreateBuffer(context, CL_MEM_READ_ONLY, dataSize, NULL, &err);
    if (err != CL_SUCCESS) goto cleanup;

    bufferB = clCreateBuffer(context, CL_MEM_READ_ONLY, dataSize, NULL, &err);
    if (err != CL_SUCCESS) goto cleanup;

    bufferResult = clCreateBuffer(context, CL_MEM_READ_WRITE, sizeof(float), NULL, &err);
    if (err != CL_SUCCESS) goto cleanup;

    // Write data
    clEnqueueWriteBuffer(queue, bufferA, CL_TRUE, 0, dataSize, dataA, 0, NULL, NULL);
    clEnqueueWriteBuffer(queue, bufferB, CL_TRUE, 0, dataSize, dataB, 0, NULL, NULL);
    clEnqueueWriteBuffer(queue, bufferResult, CL_TRUE, 0, sizeof(float), &result, 0, NULL, NULL);

    // Set kernel arguments
    clSetKernelArg(ssim_kernel, 0, sizeof(cl_mem), &bufferA);
    clSetKernelArg(ssim_kernel, 1, sizeof(cl_mem), &bufferB);
    clSetKernelArg(ssim_kernel, 2, sizeof(cl_mem), &bufferResult);
    clSetKernelArg(ssim_kernel, 3, sizeof(cl_uint), &width);
    clSetKernelArg(ssim_kernel, 4, sizeof(cl_uint), &height);

    // Execute kernel
    size_t globalSize[2] = { (size_t)width, (size_t)height };
    clEnqueueNDRangeKernel(queue, ssim_kernel, 2, NULL, globalSize, NULL, 0, NULL, NULL);

    // Read result
    clEnqueueReadBuffer(queue, bufferResult, CL_TRUE, 0, sizeof(float), &result, 0, NULL, NULL);

cleanup:
    if (bufferA) clReleaseMemObject(bufferA);
    if (bufferB) clReleaseMemObject(bufferB);
    if (bufferResult) clReleaseMemObject(bufferResult);

    if (result < 0) return -1.0f;

    float total = (float)width * height * 255.0f * 3.0f;
    return 1.0f - result / total;
}

static float computeEdgeScore(int width, int height, const uint8_t* dataA, const uint8_t* dataB) {
    cl_int err;
    cl_mem bufferA = NULL, bufferB = NULL, bufferResult = NULL;
    float result = 0.0f;

    size_t dataSize = (size_t)width * (size_t)height * 4;

    // Create buffers
    bufferA = clCreateBuffer(context, CL_MEM_READ_ONLY, dataSize, NULL, &err);
    if (err != CL_SUCCESS) goto cleanup;

    bufferB = clCreateBuffer(context, CL_MEM_READ_ONLY, dataSize, NULL, &err);
    if (err != CL_SUCCESS) goto cleanup;

    bufferResult = clCreateBuffer(context, CL_MEM_READ_WRITE, sizeof(float), NULL, &err);
    if (err != CL_SUCCESS) goto cleanup;

    // Write data
    clEnqueueWriteBuffer(queue, bufferA, CL_TRUE, 0, dataSize, dataA, 0, NULL, NULL);
    clEnqueueWriteBuffer(queue, bufferB, CL_TRUE, 0, dataSize, dataB, 0, NULL, NULL);
    clEnqueueWriteBuffer(queue, bufferResult, CL_TRUE, 0, sizeof(float), &result, 0, NULL, NULL);

    // Set kernel arguments
    clSetKernelArg(edge_kernel, 0, sizeof(cl_mem), &bufferA);
    clSetKernelArg(edge_kernel, 1, sizeof(cl_mem), &bufferB);
    clSetKernelArg(edge_kernel, 2, sizeof(cl_mem), &bufferResult);
    clSetKernelArg(edge_kernel, 3, sizeof(cl_uint), &width);
    clSetKernelArg(edge_kernel, 4, sizeof(cl_uint), &height);

    // Execute kernel
    size_t globalSize[2] = { (size_t)(width - 1), (size_t)(height - 1) };
    clEnqueueNDRangeKernel(queue, edge_kernel, 2, NULL, globalSize, NULL, 0, NULL, NULL);

    // Read result
    clEnqueueReadBuffer(queue, bufferResult, CL_TRUE, 0, sizeof(float), &result, 0, NULL, NULL);

cleanup:
    if (bufferA) clReleaseMemObject(bufferA);
    if (bufferB) clReleaseMemObject(bufferB);
    if (bufferResult) clReleaseMemObject(bufferResult);

    if (result < 0) return -1.0f;

    float total = (float)(width - 1) * (height - 1) * 255.0f * 3.0f;
    return 1.0f - result / total;
}

*/
import "C"

import (
	"errors"
	"image"
	"image/color"
	"sync"
	"unsafe"

	"github.com/zoulux/img2webp/internal/metric/gpu"
)

var (
	initOnce sync.Once
	initErr  error
)

// OpenCLBackend implements GPU acceleration using OpenCL.
type OpenCLBackend struct {
	available bool
}

// NewOpenCLBackend creates a new OpenCL GPU backend.
func NewOpenCLBackend() (*OpenCLBackend, error) {
	initOnce.Do(func() {
		if C.initOpenCL() == 0 {
			initErr = errors.New("failed to initialize OpenCL")
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return &OpenCLBackend{available: true}, nil
}

// Available returns true if OpenCL is available on this system.
func (b *OpenCLBackend) Available() bool {
	return b.available && C.isOpenCLAvailable() == 1
}

// Name returns the backend name.
func (b *OpenCLBackend) Name() string {
	return "opencl"
}

// ComputeSSIM computes SSIM-like score using OpenCL.
func (b *OpenCLBackend) ComputeSSIM(imgA, imgB image.Image) float64 {
	if imgA.Bounds() != imgB.Bounds() {
		return 0
	}

	bounds := imgA.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// For small images, CPU is faster
	if !gpu.ShouldUseGPU(bounds) {
		return computeSSIMCPU(imgA, imgB)
	}

	dataA := imageToRGBA(imgA, bounds)
	dataB := imageToRGBA(imgB, bounds)

	result := C.computeSSIM(C.int(width), C.int(height),
		(*C.uint8_t)(unsafe.Pointer(&dataA[0])),
		(*C.uint8_t)(unsafe.Pointer(&dataB[0])))

	if result < 0 {
		return computeSSIMCPU(imgA, imgB)
	}

	return float64(result)
}

// ComputeEdgeScore computes edge similarity using OpenCL.
func (b *OpenCLBackend) ComputeEdgeScore(imgA, imgB image.Image) float64 {
	if imgA.Bounds() != imgB.Bounds() {
		return 0
	}

	bounds := imgA.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if !gpu.ShouldUseGPU(bounds) {
		return computeEdgeScoreCPU(imgA, imgB)
	}

	dataA := imageToRGBA(imgA, bounds)
	dataB := imageToRGBA(imgB, bounds)

	result := C.computeEdgeScore(C.int(width), C.int(height),
		(*C.uint8_t)(unsafe.Pointer(&dataA[0])),
		(*C.uint8_t)(unsafe.Pointer(&dataB[0])))

	if result < 0 {
		return computeEdgeScoreCPU(imgA, imgB)
	}

	return float64(result)
}

// imageToRGBA converts an image to RGBA byte array
func imageToRGBA(img image.Image, bounds image.Rectangle) []byte {
	width, height := bounds.Dx(), bounds.Dy()
	data := make([]byte, width*height*4)

	// Fast path for NRGBA
	if nrgba, ok := img.(*image.NRGBA); ok {
		for y := 0; y < height; y++ {
			srcOff := y * nrgba.Stride
			dstOff := y * width * 4
			copy(data[dstOff:dstOff+width*4], nrgba.Pix[srcOff:srcOff+width*4])
		}
		return data
	}

	// Fast path for RGBA
	if rgba, ok := img.(*image.RGBA); ok {
		for y := 0; y < height; y++ {
			srcOff := y * rgba.Stride
			dstOff := y * width * 4
			copy(data[dstOff:dstOff+width*4], rgba.Pix[srcOff:srcOff+width*4])
		}
		return data
	}

	// Fast path for YCbCr (JPEG)
	if ycbcr, ok := img.(*image.YCbCr); ok {
		idx := 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				yi := ycbcr.Y[y*ycbcr.YStride+x]
				data[idx] = yi
				data[idx+1] = yi
				data[idx+2] = yi
				data[idx+3] = 255
				idx += 4
			}
		}
		return data
	}

	// Fast path for NYCbCrA (WebP)
	if nycbcra, ok := img.(*image.NYCbCrA); ok {
		idx := 0
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				yi := nycbcra.Y[y*nycbcra.YStride+x]
				data[idx] = yi
				data[idx+1] = yi
				data[idx+2] = yi
				data[idx+3] = 255
				idx += 4
			}
		}
		return data
	}

	// Fallback
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.NRGBAModel.Convert(img.At(x+bounds.Min.X, y+bounds.Min.Y)).(color.NRGBA)
			idx := (y*width + x) * 4
			data[idx] = c.R
			data[idx+1] = c.G
			data[idx+2] = c.B
			data[idx+3] = c.A
		}
	}

	return data
}

// computeSSIMCPU computes SSIM-like score on CPU (fallback).
func computeSSIMCPU(a, b image.Image) float64 {
	bounds := a.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	var totalDiff float64

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ca := color.NRGBAModel.Convert(a.At(x, y)).(color.NRGBA)
			cb := color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA)
			totalDiff += float64(abs(int(ca.R)-int(cb.R))) +
				float64(abs(int(ca.G)-int(cb.G))) +
				float64(abs(int(ca.B)-int(cb.B)))
		}
	}

	total := float64(width) * float64(height) * 255.0 * 3.0
	return 1.0 - totalDiff/total
}

// computeEdgeScoreCPU computes edge similarity on CPU (fallback).
func computeEdgeScoreCPU(a, b image.Image) float64 {
	bounds := a.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	var totalDiff float64

	for y := bounds.Min.Y; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X; x < bounds.Max.X-1; x++ {
			ca1 := color.NRGBAModel.Convert(a.At(x, y)).(color.NRGBA)
			ca2 := color.NRGBAModel.Convert(a.At(x+1, y)).(color.NRGBA)
			cb1 := color.NRGBAModel.Convert(b.At(x, y)).(color.NRGBA)
			cb2 := color.NRGBAModel.Convert(b.At(x+1, y)).(color.NRGBA)

			edgeA := float64(abs(int(ca1.R)-int(ca2.R))) +
				float64(abs(int(ca1.G)-int(ca2.G))) +
				float64(abs(int(ca1.B)-int(ca2.B)))
			edgeB := float64(abs(int(cb1.R)-int(cb2.R))) +
				float64(abs(int(cb1.G)-int(cb2.G))) +
				float64(abs(int(cb1.B)-int(cb2.B)))

			totalDiff += absFloat(edgeA - edgeB)
		}
	}

	total := float64(width-1) * float64(height-1) * 255.0 * 3.0
	return 1.0 - totalDiff/total
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
