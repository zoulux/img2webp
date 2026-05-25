//go:build darwin && arm64

// Package metal provides Metal GPU acceleration for metric computation on macOS.
package metal

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Metal -framework Foundation -framework CoreGraphics

#import <Metal/Metal.h>
#import <CoreGraphics/CoreGraphics.h>
#import <simd/simd.h>

// Metal shader source code
static const char* shaderSource = R"(
#include <metal_stdlib>
#include <simd/simd.h>
using namespace metal;

// SSIM compute kernel - computes per-pixel difference
kernel void ssim_kernel(
    texture2d<float, access::read> imgA [[texture(0)]],
    texture2d<float, access::read> imgB [[texture(1)]],
    device atomic_float* totalDiff [[buffer(0)]],
    uint2 gid [[thread_position_in_grid]]
) {
    if (gid.x >= imgA.get_width() || gid.y >= imgA.get_height()) return;

    float4 a = imgA.read(gid);
    float4 b = imgB.read(gid);

    float diff = abs(a.r - b.r) + abs(a.g - b.g) + abs(a.b - b.b);
    atomic_fetch_add_explicit(totalDiff, diff, memory_order_relaxed);
}

// Edge score compute kernel
kernel void edge_kernel(
    texture2d<float, access::read> imgA [[texture(0)]],
    texture2d<float, access::read> imgB [[texture(1)]],
    device atomic_float* totalDiff [[buffer(0)]],
    uint2 gid [[thread_position_in_grid]]
) {
    uint width = imgA.get_width();
    uint height = imgA.get_height();

    if (gid.x >= width - 1 || gid.y >= height - 1) return;

    float4 a1 = imgA.read(gid);
    float4 a2 = imgA.read(uint2(gid.x + 1, gid.y));
    float4 b1 = imgB.read(gid);
    float4 b2 = imgB.read(uint2(gid.x + 1, gid.y));

    float edgeA = abs(a1.r - a2.r) + abs(a1.g - a2.g) + abs(a1.b - a2.b);
    float edgeB = abs(b1.r - b2.r) + abs(b1.g - b2.g) + abs(b1.b - b2.b);

    float diff = abs(edgeA - edgeB);
    atomic_fetch_add_explicit(totalDiff, diff, memory_order_relaxed);
}
)";

static id<MTLDevice> device = nil;
static id<MTLCommandQueue> commandQueue = nil;
static id<MTLLibrary> library = nil;
static id<MTLComputePipelineState> ssimPipeline = nil;
static id<MTLComputePipelineState> edgePipeline = nil;

static int initMetal() {
    @autoreleasepool {
        device = MTLCreateSystemDefaultDevice();
        if (!device) return 0;

        commandQueue = [device newCommandQueue];
        if (!commandQueue) return 0;

        NSError* error = nil;
        NSString* source = [NSString stringWithUTF8String:shaderSource];
        MTLCompileOptions* options = [[MTLCompileOptions alloc] init];

        library = [device newLibraryWithSource:source options:options error:&error];
        if (!library) {
            NSLog(@"Failed to compile Metal library: %@", error);
            return 0;
        }

        id<MTLFunction> ssimFunction = [library newFunctionWithName:@"ssim_kernel"];
        if (!ssimFunction) return 0;

        id<MTLFunction> edgeFunction = [library newFunctionWithName:@"edge_kernel"];
        if (!edgeFunction) return 0;

        ssimPipeline = [device newComputePipelineStateWithFunction:ssimFunction error:&error];
        if (!ssimPipeline) return 0;

        edgePipeline = [device newComputePipelineStateWithFunction:edgeFunction error:&error];
        if (!edgePipeline) return 0;

        return 1;
    }
}

static id<MTLTexture> createTexture(int width, int height, const uint8_t* data) {
    @autoreleasepool {
        MTLTextureDescriptor* desc = [[MTLTextureDescriptor alloc] init];
        desc.textureType = MTLTextureType2D;
        desc.pixelFormat = MTLPixelFormatRGBA8Unorm;
        desc.width = width;
        desc.height = height;
        desc.usage = MTLTextureUsageShaderRead;

        id<MTLTexture> texture = [device newTextureWithDescriptor:desc];
        if (!texture) return nil;

        MTLRegion region = MTLRegionMake2D(0, 0, width, height);
        [texture replaceRegion:region mipmapLevel:0 withBytes:data bytesPerRow:width * 4];

        return texture;
    }
}

static float computeSSIM(int width, int height, const uint8_t* dataA, const uint8_t* dataB) {
    @autoreleasepool {
        id<MTLTexture> texA = createTexture(width, height, dataA);
        id<MTLTexture> texB = createTexture(width, height, dataB);
        if (!texA || !texB) return -1.0f;

        id<MTLBuffer> resultBuffer = [device newBufferWithLength:sizeof(float) options:MTLResourceStorageModeShared];
        float* resultPtr = (float*)resultBuffer.contents;
        *resultPtr = 0.0f;

        id<MTLCommandBuffer> cmdBuffer = [commandQueue commandBuffer];
        id<MTLComputeCommandEncoder> encoder = [cmdBuffer computeCommandEncoder];

        [encoder setComputePipelineState:ssimPipeline];
        [encoder setTexture:texA atIndex:0];
        [encoder setTexture:texB atIndex:1];
        [encoder setBuffer:resultBuffer offset:0 atIndex:0];

        NSUInteger w = MAX(1, (NSUInteger)width);
        NSUInteger h = MAX(1, (NSUInteger)height);
        MTLSize gridSize = MTLSizeMake(w, h, 1);

        NSUInteger threadWidth = MIN(ssimPipeline.maxTotalThreadsPerThreadgroup, w);
        NSUInteger threadHeight = MIN(ssimPipeline.maxTotalThreadsPerThreadgroup / threadWidth, h);
        threadWidth = MAX(1, threadWidth);
        threadHeight = MAX(1, threadHeight);
        MTLSize threadGroupSize = MTLSizeMake(threadWidth, threadHeight, 1);

        [encoder dispatchThreads:gridSize threadsPerThreadgroup:threadGroupSize];
        [encoder endEncoding];

        [cmdBuffer commit];
        [cmdBuffer waitUntilCompleted];

        // Metal texture values are normalized (0-1), multiply by 255 to match CPU
        float totalDiff = *resultPtr * 255.0f;
        float total = (float)width * height * 255.0f * 3.0f;
        return 1.0f - totalDiff / total;
    }
}

static float computeEdgeScore(int width, int height, const uint8_t* dataA, const uint8_t* dataB) {
    @autoreleasepool {
        id<MTLTexture> texA = createTexture(width, height, dataA);
        id<MTLTexture> texB = createTexture(width, height, dataB);
        if (!texA || !texB) return -1.0f;

        id<MTLBuffer> resultBuffer = [device newBufferWithLength:sizeof(float) options:MTLResourceStorageModeShared];
        float* resultPtr = (float*)resultBuffer.contents;
        *resultPtr = 0.0f;

        id<MTLCommandBuffer> cmdBuffer = [commandQueue commandBuffer];
        id<MTLComputeCommandEncoder> encoder = [cmdBuffer computeCommandEncoder];

        [encoder setComputePipelineState:edgePipeline];
        [encoder setTexture:texA atIndex:0];
        [encoder setTexture:texB atIndex:1];
        [encoder setBuffer:resultBuffer offset:0 atIndex:0];

        NSUInteger w = MAX(1, (NSUInteger)(width - 1));
        NSUInteger h = MAX(1, (NSUInteger)(height - 1));
        MTLSize gridSize = MTLSizeMake(w, h, 1);

        NSUInteger threadWidth = MIN(edgePipeline.maxTotalThreadsPerThreadgroup, w);
        NSUInteger threadHeight = MIN(edgePipeline.maxTotalThreadsPerThreadgroup / threadWidth, h);
        threadWidth = MAX(1, threadWidth);
        threadHeight = MAX(1, threadHeight);
        MTLSize threadGroupSize = MTLSizeMake(threadWidth, threadHeight, 1);

        [encoder dispatchThreads:gridSize threadsPerThreadgroup:threadGroupSize];
        [encoder endEncoding];

        [cmdBuffer commit];
        [cmdBuffer waitUntilCompleted];

        // Metal texture values are normalized (0-1), multiply by 255 to match CPU
        float totalDiff = *resultPtr * 255.0f;
        float total = (float)(width - 1) * (height - 1) * 255.0f * 3.0f;
        return 1.0f - totalDiff / total;
    }
}

static int isMetalAvailable() {
    return device != nil ? 1 : 0;
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

// MetalBackend implements GPU acceleration using Metal.
type MetalBackend struct {
	available bool
}

// NewMetalBackend creates a new Metal GPU backend.
func NewMetalBackend() (*MetalBackend, error) {
	initOnce.Do(func() {
		if C.initMetal() == 0 {
			initErr = errors.New("failed to initialize Metal")
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return &MetalBackend{available: true}, nil
}

// Available returns true if Metal is available on this system.
func (m *MetalBackend) Available() bool {
	return m.available && C.isMetalAvailable() == 1
}

// Name returns the backend name.
func (m *MetalBackend) Name() string {
	return "metal"
}

// ComputeSSIM computes SSIM-like score using Metal.
func (m *MetalBackend) ComputeSSIM(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 0
	}

	bounds := a.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	// For small images, CPU is faster
	if !gpu.ShouldUseGPU(bounds) {
		return computeSSIMCPU(a, b)
	}

	dataA := imageToRGBA(a, bounds)
	dataB := imageToRGBA(b, bounds)

	result := C.computeSSIM(C.int(width), C.int(height),
		(*C.uint8_t)(unsafe.Pointer(&dataA[0])),
		(*C.uint8_t)(unsafe.Pointer(&dataB[0])))

	if result < 0 {
		return computeSSIMCPU(a, b)
	}

	return float64(result)
}

// ComputeEdgeScore computes edge similarity using Metal.
func (m *MetalBackend) ComputeEdgeScore(a, b image.Image) float64 {
	if a.Bounds() != b.Bounds() {
		return 0
	}

	bounds := a.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	if !gpu.ShouldUseGPU(bounds) {
		return computeEdgeScoreCPU(a, b)
	}

	dataA := imageToRGBA(a, bounds)
	dataB := imageToRGBA(b, bounds)

	result := C.computeEdgeScore(C.int(width), C.int(height),
		(*C.uint8_t)(unsafe.Pointer(&dataA[0])),
		(*C.uint8_t)(unsafe.Pointer(&dataB[0])))

	if result < 0 {
		return computeEdgeScoreCPU(a, b)
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
				data[idx] = ycbcr.Y[y*ycbcr.YStride+x] // Use Y channel only
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

// CPU fallback implementations
func computeSSIMCPU(a, b image.Image) float64 {
	bounds := a.Bounds()
	var diff float64
	var total float64
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			diff += abs64(float64(int(ar>>8) - int(br>>8)))
			diff += abs64(float64(int(ag>>8) - int(bg>>8)))
			diff += abs64(float64(int(ab>>8) - int(bb>>8)))
			total += 255 * 3
		}
	}
	if total == 0 {
		return 1
	}
	return 1 - diff/total
}

func computeEdgeScoreCPU(a, b image.Image) float64 {
	bounds := a.Bounds()
	var diff float64
	var total float64
	for y := bounds.Min.Y; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X; x < bounds.Max.X-1; x++ {
			ad := edgeMagnitude(a, x, y)
			bd := edgeMagnitude(b, x, y)
			diff += abs64(ad - bd)
			total += 255 * 3
		}
	}
	if total == 0 {
		return 1
	}
	return 1 - diff/total
}

func edgeMagnitude(img image.Image, x, y int) float64 {
	r1, g1, b1, _ := img.At(x, y).RGBA()
	r2, g2, b2, _ := img.At(x+1, y).RGBA()
	return abs64(float64(int(r1>>8)-int(r2>>8))) + abs64(float64(int(g1>>8)-int(g2>>8))) + abs64(float64(int(b1>>8)-int(b2>>8)))
}

func abs64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
