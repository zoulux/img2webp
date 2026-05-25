//go:build (linux || windows) && cgo

package cuda

/*
#cgo linux LDFLAGS: -ldl
#cgo windows LDFLAGS: -lkernel32

#include <stdlib.h>
#include <string.h>
#include <stdint.h>

#ifdef __linux__
#include <dlfcn.h>
#define CUDA_LIB "libcuda.so"
#define LOAD_LIB(name) dlopen(name, RTLD_LAZY)
#define GET_SYM(handle, name) dlsym(handle, name)
#define CLOSE_LIB(handle) dlclose(handle)
#else
#include <windows.h>
#define CUDA_LIB "nvcuda.dll"
#define LOAD_LIB(name) LoadLibraryA(name)
#define GET_SYM(handle, name) GetProcAddress((HMODULE)handle, name)
#define CLOSE_LIB(handle) FreeLibrary((HMODULE)handle)
#endif

// CUDA types
typedef int CUresult;
typedef void* CUdeviceptr;
typedef int CUdevice;
typedef void* CUcontext;
typedef void* CUmodule;
typedef void* CUfunction;
typedef void* CUstream;

#define CU_SUCCESS 0

// CUDA function pointer types
typedef CUresult (*cuInit_t)(unsigned int);
typedef CUresult (*cuDeviceGet_t)(CUdevice*, int);
typedef CUresult (*cuDeviceGetCount_t)(int*);
typedef CUresult (*cuDeviceGetName_t)(char*, int, CUdevice);
typedef CUresult (*cuCtxCreate_t)(CUcontext*, unsigned int, CUdevice);
typedef CUresult (*cuCtxDestroy_t)(CUcontext);
typedef CUresult (*cuCtxSynchronize_t)(void);
typedef CUresult (*cuMemAlloc_t)(CUdeviceptr*, size_t);
typedef CUresult (*cuMemFree_t)(CUdeviceptr);
typedef CUresult (*cuMemcpyHtoD_t)(CUdeviceptr, const void*, size_t);
typedef CUresult (*cuMemcpyDtoH_t)(void*, CUdeviceptr, size_t);
typedef CUresult (*cuModuleLoadData_t)(CUmodule*, const void*);
typedef CUresult (*cuModuleUnload_t)(CUmodule);
typedef CUresult (*cuModuleGetFunction_t)(CUfunction*, CUmodule, const char*);
typedef CUresult (*cuLaunchKernel_t)(CUfunction, unsigned int, unsigned int, unsigned int,
                                      unsigned int, unsigned int, unsigned int,
                                      unsigned int, CUstream, void**, void**);

// Global CUDA resources
static void* cudaLib = NULL;
static CUdevice device = -1;
static CUcontext context = NULL;
static CUmodule module = NULL;
static CUfunction ssimFunc = NULL;
static CUfunction edgeFunc = NULL;

// Function pointers
static cuInit_t cuInit_ptr = NULL;
static cuDeviceGet_t cuDeviceGet_ptr = NULL;
static cuDeviceGetCount_t cuDeviceGetCount_ptr = NULL;
static cuDeviceGetName_t cuDeviceGetName_ptr = NULL;
static cuCtxCreate_t cuCtxCreate_ptr = NULL;
static cuCtxDestroy_t cuCtxDestroy_ptr = NULL;
static cuCtxSynchronize_t cuCtxSynchronize_ptr = NULL;
static cuMemAlloc_t cuMemAlloc_ptr = NULL;
static cuMemFree_t cuMemFree_ptr = NULL;
static cuMemcpyHtoD_t cuMemcpyHtoD_ptr = NULL;
static cuMemcpyDtoH_t cuMemcpyDtoH_ptr = NULL;
static cuModuleLoadData_t cuModuleLoadData_ptr = NULL;
static cuModuleUnload_t cuModuleUnload_ptr = NULL;
static cuModuleGetFunction_t cuModuleGetFunction_ptr = NULL;
static cuLaunchKernel_t cuLaunchKernel_ptr = NULL;

// CUDA PTX kernel source for SSIM and Edge detection
// This is compiled at runtime by the CUDA driver
static const char* ptxSource =
".version 7.5\n"
".target sm_50\n"
".address_size 64\n"
"\n"
// SSIM kernel - computes per-pixel RGB difference
".visible .entry ssim_kernel(\n"
"    .param .u64 imgA,\n"
"    .param .u64 imgB,\n"
"    .param .u64 totalDiff,\n"
"    .param .u32 width,\n"
"    .param .u32 height\n"
") {\n"
"    .reg .u32 tid_x, tid_y, blk_x, blk_y, x, y;\n"
"    .reg .u32 width_r, height_r, idx;\n"
"    .reg .u64 imgA_r, imgB_r, totalDiff_r, addr;\n"
"    .reg .u8 a_r, a_g, a_b, b_r, b_g, b_b;\n"
"    .reg .s32 diff_r, diff_g, diff_b, diff;\n"
"    .reg .f32 diff_f;\n"
"\n"
"    // Load parameters\n"
"    ld.param.u64 imgA_r, [imgA];\n"
"    ld.param.u64 imgB_r, [imgB];\n"
"    ld.param.u64 totalDiff_r, [totalDiff];\n"
"    ld.param.u32 width_r, [width];\n"
"    ld.param.u32 height_r, [height];\n"
"\n"
"    // Calculate global thread ID\n"
"    mov.u32 tid_x, %tid.x;\n"
"    mov.u32 tid_y, %tid.y;\n"
"    mov.u32 blk_x, %ctaid.x;\n"
"    mov.u32 blk_y, %ctaid.y;\n"
"    mad.lo.u32 x, blk_x, 16, tid_x;\n"
"    mad.lo.u32 y, blk_y, 16, tid_y;\n"
"\n"
"    // Bounds check\n"
"    setp.ge.u32 p0, x, width_r;\n"
"    setp.ge.u32 p1, y, height_r;\n"
"    @p0 bra exit;\n"
"    @p1 bra exit;\n"
"\n"
"    // Calculate pixel index (RGBA = 4 bytes per pixel)\n"
"    mad.lo.u32 idx, y, width_r, x;\n"
"    shl.u32 idx, idx, 2;\n"
"\n"
"    // Load pixel A\n"
"    add.u64 addr, imgA_r, idx;\n"
"    ld.u8 a_r, [addr];\n"
"    ld.u8 a_g, [addr+1];\n"
"    ld.u8 a_b, [addr+2];\n"
"\n"
"    // Load pixel B\n"
"    add.u64 addr, imgB_r, idx;\n"
"    ld.u8 b_r, [addr];\n"
"    ld.u8 b_g, [addr+1];\n"
"    ld.u8 b_b, [addr+2];\n"
"\n"
"    // Compute absolute differences\n"
"    sub.s16 diff_r, a_r, b_r;\n"
"    abs.s16 diff_r, diff_r;\n"
"    sub.s16 diff_g, a_g, b_g;\n"
"    abs.s16 diff_g, diff_g;\n"
"    sub.s16 diff_b, a_b, b_b;\n"
"    abs.s16 diff_b, diff_b;\n"
"\n"
"    // Sum differences\n"
"    add.s32 diff, diff_r, diff_g;\n"
"    add.s32 diff, diff, diff_b;\n"
"\n"
"    // Convert to float and atomic add\n"
"    cvt.f32.s32 diff_f, diff;\n"
"    red.add.f32 totalDiff_r, [totalDiff_r], diff_f;\n"
"\n"
"exit:\n"
"    ret;\n"
"}\n"
"\n"
// Edge kernel - computes horizontal edge difference
".visible .entry edge_kernel(\n"
"    .param .u64 imgA,\n"
"    .param .u64 imgB,\n"
"    .param .u64 totalDiff,\n"
"    .param .u32 width,\n"
"    .param .u32 height\n"
") {\n"
"    .reg .u32 tid_x, tid_y, blk_x, blk_y, x, y;\n"
"    .reg .u32 width_r, height_r, idx1, idx2;\n"
"    .reg .u64 imgA_r, imgB_r, totalDiff_r, addr1, addr2;\n"
"    .reg .u8 a1_r, a1_g, a1_b, a2_r, a2_g, a2_b;\n"
"    .reg .u8 b1_r, b1_g, b1_b, b2_r, b2_g, b2_b;\n"
"    .reg .s32 edgeA, edgeB, diff;\n"
"    .reg .f32 diff_f;\n"
"\n"
"    ld.param.u64 imgA_r, [imgA];\n"
"    ld.param.u64 imgB_r, [imgB];\n"
"    ld.param.u64 totalDiff_r, [totalDiff];\n"
"    ld.param.u32 width_r, [width];\n"
"    ld.param.u32 height_r, [height];\n"
"\n"
"    mov.u32 tid_x, %tid.x;\n"
"    mov.u32 tid_y, %tid.y;\n"
"    mov.u32 blk_x, %ctaid.x;\n"
"    mov.u32 blk_y, %ctaid.y;\n"
"    mad.lo.u32 x, blk_x, 16, tid_x;\n"
"    mad.lo.u32 y, blk_y, 16, tid_y;\n"
"\n"
"    // Bounds check (need width-1 and height-1)\n"
"    setp.ge.u32 p0, x, width_r;\n"
"    setp.ge.u32 p1, y, height_r;\n"
"    @p0 bra exit;\n"
"    @p1 bra exit;\n"
"\n"
"    // Calculate pixel indices\n"
"    mad.lo.u32 idx1, y, width_r, x;\n"
"    shl.u32 idx1, idx1, 2;\n"
"    add.u32 idx2, idx1, 4;\n"
"\n"
"    // Load edge for image A (horizontal gradient)\n"
"    add.u64 addr1, imgA_r, idx1;\n"
"    add.u64 addr2, imgA_r, idx2;\n"
"    ld.u8 a1_r, [addr1]; ld.u8 a1_g, [addr1+1]; ld.u8 a1_b, [addr1+2];\n"
"    ld.u8 a2_r, [addr2]; ld.u8 a2_g, [addr2+1]; ld.u8 a2_b, [addr2+2];\n"
"    sub.s16 diff_r, a1_r, a2_r; abs.s16 diff_r, diff_r;\n"
"    sub.s16 diff_g, a1_g, a2_g; abs.s16 diff_g, diff_g;\n"
"    sub.s16 diff_b, a1_b, a2_b; abs.s16 diff_b, diff_b;\n"
"    add.s32 edgeA, diff_r, diff_g;\n"
"    add.s32 edgeA, edgeA, diff_b;\n"
"\n"
"    // Load edge for image B\n"
"    add.u64 addr1, imgB_r, idx1;\n"
"    add.u64 addr2, imgB_r, idx2;\n"
"    ld.u8 b1_r, [addr1]; ld.u8 b1_g, [addr1+1]; ld.u8 b1_b, [addr1+2];\n"
"    ld.u8 b2_r, [addr2]; ld.u8 b2_g, [addr2+1]; ld.u8 b2_b, [addr2+2];\n"
"    sub.s16 diff_r, b1_r, b2_r; abs.s16 diff_r, diff_r;\n"
"    sub.s16 diff_g, b1_g, b2_g; abs.s16 diff_g, diff_g;\n"
"    sub.s16 diff_b, b1_b, b2_b; abs.s16 diff_b, diff_b;\n"
"    add.s32 edgeB, diff_r, diff_g;\n"
"    add.s32 edgeB, edgeB, diff_b;\n"
"\n"
"    // Compute edge difference\n"
"    sub.s32 diff, edgeA, edgeB;\n"
"    abs.s32 diff, diff;\n"
"\n"
"    cvt.f32.s32 diff_f, diff;\n"
"    red.add.f32 totalDiff_r, [totalDiff_r], diff_f;\n"
"\n"
"exit:\n"
"    ret;\n"
"}\n";

// Load all CUDA function pointers
static int loadCUDASymbols() {
    #define LOAD_SYMBOL(name) \
        name##_ptr = (name##_t)GET_SYM(cudaLib, #name); \
        if (!name##_ptr) return 0

    LOAD_SYMBOL(cuInit);
    LOAD_SYMBOL(cuDeviceGet);
    LOAD_SYMBOL(cuDeviceGetCount);
    LOAD_SYMBOL(cuDeviceGetName);
    LOAD_SYMBOL(cuCtxCreate);
    LOAD_SYMBOL(cuCtxDestroy);
    LOAD_SYMBOL(cuCtxSynchronize);
    LOAD_SYMBOL(cuMemAlloc);
    LOAD_SYMBOL(cuMemFree);
    LOAD_SYMBOL(cuMemcpyHtoD);
    LOAD_SYMBOL(cuMemcpyDtoH);
    LOAD_SYMBOL(cuModuleLoadData);
    LOAD_SYMBOL(cuModuleUnload);
    LOAD_SYMBOL(cuModuleGetFunction);
    LOAD_SYMBOL(cuLaunchKernel);

    #undef LOAD_SYMBOL
    return 1;
}

// Initialize CUDA
static int initCUDA() {
    // 1. Load CUDA library
    cudaLib = LOAD_LIB(CUDA_LIB);
    if (!cudaLib) return 0;

    // 2. Load function pointers
    if (!loadCUDASymbols()) {
        CLOSE_LIB(cudaLib);
        cudaLib = NULL;
        return 0;
    }

    // 3. Initialize CUDA
    if (cuInit_ptr(0) != CU_SUCCESS) {
        CLOSE_LIB(cudaLib);
        cudaLib = NULL;
        return 0;
    }

    // 4. Get device count
    int deviceCount = 0;
    if (cuDeviceGetCount_ptr(&deviceCount) != CU_SUCCESS || deviceCount == 0) {
        CLOSE_LIB(cudaLib);
        cudaLib = NULL;
        return 0;
    }

    // 5. Get first device
    if (cuDeviceGet_ptr(&device, 0) != CU_SUCCESS) {
        CLOSE_LIB(cudaLib);
        cudaLib = NULL;
        return 0;
    }

    // 6. Create context
    if (cuCtxCreate_ptr(&context, 0, device) != CU_SUCCESS) {
        CLOSE_LIB(cudaLib);
        cudaLib = NULL;
        device = -1;
        return 0;
    }

    // 7. Load PTX module
    if (cuModuleLoadData_ptr(&module, ptxSource) != CU_SUCCESS) {
        cuCtxDestroy_ptr(context);
        CLOSE_LIB(cudaLib);
        context = NULL;
        cudaLib = NULL;
        device = -1;
        return 0;
    }

    // 8. Get kernel functions
    if (cuModuleGetFunction_ptr(&ssimFunc, module, "ssim_kernel") != CU_SUCCESS) {
        cuModuleUnload_ptr(module);
        cuCtxDestroy_ptr(context);
        CLOSE_LIB(cudaLib);
        ssimFunc = NULL;
        module = NULL;
        context = NULL;
        cudaLib = NULL;
        device = -1;
        return 0;
    }

    if (cuModuleGetFunction_ptr(&edgeFunc, module, "edge_kernel") != CU_SUCCESS) {
        cuModuleUnload_ptr(module);
        cuCtxDestroy_ptr(context);
        CLOSE_LIB(cudaLib);
        ssimFunc = NULL;
        module = NULL;
        context = NULL;
        cudaLib = NULL;
        device = -1;
        return 0;
    }

    return 1;
}

// Check if CUDA is available
static int isCUDAAvailable() {
    return (cudaLib != NULL && context != NULL &&
            ssimFunc != NULL && edgeFunc != NULL) ? 1 : 0;
}

// Cleanup CUDA resources
static void cleanupCUDA() {
    if (ssimFunc) { ssimFunc = NULL; }
    if (edgeFunc) { edgeFunc = NULL; }
    if (module) {
        cuModuleUnload_ptr(module);
        module = NULL;
    }
    if (context) {
        cuCtxDestroy_ptr(context);
        context = NULL;
    }
    if (cudaLib) {
        CLOSE_LIB(cudaLib);
        cudaLib = NULL;
    }
    device = -1;
}

// Compute SSIM using CUDA
static float computeSSIM(int width, int height, const uint8_t* dataA, const uint8_t* dataB) {
    CUdeviceptr d_A = 0, d_B = 0, d_result = 0;
    float result = 0.0f;

    size_t dataSize = (size_t)width * (size_t)height * 4;

    // Allocate device memory
    if (cuMemAlloc_ptr(&d_A, dataSize) != CU_SUCCESS) goto cleanup;
    if (cuMemAlloc_ptr(&d_B, dataSize) != CU_SUCCESS) goto cleanup;
    if (cuMemAlloc_ptr(&d_result, sizeof(float)) != CU_SUCCESS) goto cleanup;

    // Copy data to device
    cuMemcpyHtoD_ptr(d_A, dataA, dataSize);
    cuMemcpyHtoD_ptr(d_B, dataB, dataSize);
    cuMemcpyHtoD_ptr(d_result, &result, sizeof(float));

    // Set kernel arguments
    void* args[] = { &d_A, &d_B, &d_result, &width, &height };

    // Launch kernel with 16x16 thread blocks
    int blockSize = 16;
    int gridSizeX = (width + blockSize - 1) / blockSize;
    int gridSizeY = (height + blockSize - 1) / blockSize;

    if (cuLaunchKernel_ptr(ssimFunc, gridSizeX, gridSizeY, 1,
                           blockSize, blockSize, 1,
                           0, NULL, args, NULL) != CU_SUCCESS) {
        goto cleanup;
    }

    // Synchronize and read result
    cuCtxSynchronize_ptr();
    cuMemcpyDtoH_ptr(&result, d_result, sizeof(float));

cleanup:
    if (d_A) cuMemFree_ptr(d_A);
    if (d_B) cuMemFree_ptr(d_B);
    if (d_result) cuMemFree_ptr(d_result);

    if (result < 0) return -1.0f;

    float total = (float)width * height * 255.0f * 3.0f;
    return 1.0f - result / total;
}

// Compute Edge Score using CUDA
static float computeEdgeScore(int width, int height, const uint8_t* dataA, const uint8_t* dataB) {
    CUdeviceptr d_A = 0, d_B = 0, d_result = 0;
    float result = 0.0f;

    size_t dataSize = (size_t)width * (size_t)height * 4;

    if (cuMemAlloc_ptr(&d_A, dataSize) != CU_SUCCESS) goto cleanup;
    if (cuMemAlloc_ptr(&d_B, dataSize) != CU_SUCCESS) goto cleanup;
    if (cuMemAlloc_ptr(&d_result, sizeof(float)) != CU_SUCCESS) goto cleanup;

    cuMemcpyHtoD_ptr(d_A, dataA, dataSize);
    cuMemcpyHtoD_ptr(d_B, dataB, dataSize);
    cuMemcpyHtoD_ptr(d_result, &result, sizeof(float));

    void* args[] = { &d_A, &d_B, &d_result, &width, &height };

    int blockSize = 16;
    int gridSizeX = (width - 1 + blockSize - 1) / blockSize;
    int gridSizeY = (height - 1 + blockSize - 1) / blockSize;
    if (gridSizeX < 1) gridSizeX = 1;
    if (gridSizeY < 1) gridSizeY = 1;

    if (cuLaunchKernel_ptr(edgeFunc, gridSizeX, gridSizeY, 1,
                           blockSize, blockSize, 1,
                           0, NULL, args, NULL) != CU_SUCCESS) {
        goto cleanup;
    }

    cuCtxSynchronize_ptr();
    cuMemcpyDtoH_ptr(&result, d_result, sizeof(float));

cleanup:
    if (d_A) cuMemFree_ptr(d_A);
    if (d_B) cuMemFree_ptr(d_B);
    if (d_result) cuMemFree_ptr(d_result);

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

// CUDABackend implements GPU acceleration using CUDA.
type CUDABackend struct {
	available bool
}

// NewCUDABackend creates a new CUDA GPU backend.
func NewCUDABackend() (*CUDABackend, error) {
	initOnce.Do(func() {
		if C.initCUDA() == 0 {
			initErr = errors.New("failed to initialize CUDA")
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return &CUDABackend{available: true}, nil
}

// Available returns true if CUDA is available on this system.
func (b *CUDABackend) Available() bool {
	return b.available && C.isCUDAAvailable() == 1
}

// Name returns the backend name.
func (b *CUDABackend) Name() string {
	return "cuda"
}

// ComputeSSIM computes SSIM-like score using CUDA.
func (b *CUDABackend) ComputeSSIM(imgA, imgB image.Image) float64 {
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

// ComputeEdgeScore computes edge similarity using CUDA.
func (b *CUDABackend) ComputeEdgeScore(imgA, imgB image.Image) float64 {
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
