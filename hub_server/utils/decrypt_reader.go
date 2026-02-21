package utils

import (
	"encoding/binary"
	"io"
)

// DecryptReader 是一个支持流式解密的 io.Reader 包装器
// 它使用 ISAAC64 算法生成密钥流，并对读取的数据进行 XOR 解密
// 支持 Range 请求，可以从任意偏移位置开始解密
type DecryptReader struct {
	reader   io.Reader   // 底层数据源
	ctx      *Isaac64Ctx // ISAAC64 上下文
	limit    uint64      // 加密区域大小（字节）
	consumed uint64      // 已处理的字节数
	ks       [8]byte     // 当前密钥块（8字节）
	ksPos    int         // 密钥块中的当前位置
}

// Isaac64Ctx 是 ISAAC64 伪随机数生成器的上下文
type Isaac64Ctx struct {
	randrsl [256]uint64
	randcnt uint64
	mm      [256]uint64
	aa      uint64
	bb      uint64
	cc      uint64
}

// NewDecryptReader 创建一个新的解密读取器
func NewDecryptReader(reader io.Reader, key uint64, offset uint64, limit uint64) *DecryptReader {
	ctx := newIsaac64Context(key)
	dr := &DecryptReader{
		reader:   reader,
		ctx:      ctx,
		limit:    limit,
		consumed: 0,
		ksPos:    8,
	}

	if limit > 0 {
		if offset >= limit {
			dr.consumed = limit
		} else {
			dr.consumed = offset
			skipBlocks := offset / 8
			for i := uint64(0); i < skipBlocks; i++ {
				_ = dr.ctx.isaac64Random()
			}
			randNumber := dr.ctx.isaac64Random()
			binary.BigEndian.PutUint64(dr.ks[:], randNumber)
			dr.ksPos = int(offset % 8)
		}
	}
	return dr
}

// Read 实现 io.Reader 接口
func (dr *DecryptReader) Read(p []byte) (int, error) {
	n, err := dr.reader.Read(p)
	if n <= 0 {
		return n, err
	}

	if dr.limit == 0 || dr.consumed >= dr.limit {
		return n, err
	}

	toDecrypt := uint64(n)
	remaining := dr.limit - dr.consumed
	if toDecrypt > remaining {
		toDecrypt = remaining
	}

	for i := uint64(0); i < toDecrypt; i++ {
		if dr.ksPos >= 8 {
			randNumber := dr.ctx.isaac64Random()
			binary.BigEndian.PutUint64(dr.ks[:], randNumber)
			dr.ksPos = 0
		}
		p[i] ^= dr.ks[dr.ksPos]
		dr.ksPos++
	}
	dr.consumed += toDecrypt
	return n, err
}

// newIsaac64Context 创建并初始化 ISAAC64 上下文
func newIsaac64Context(seed uint64) *Isaac64Ctx {
	ctx := &Isaac64Ctx{}
	ctx.randrsl[0] = seed
	ctx.randinit(true)
	return ctx
}

func (ctx *Isaac64Ctx) randinit(flag bool) {
	var a, b, c, d, e, f, g, h uint64
	a = 0x9e3779b97f4a7c13
	b, c, d, e, f, g, h = a, a, a, a, a, a, a

	for j := 0; j < 4; j++ {
		a, b, c, d, e, f, g, h = ctx.mix(a, b, c, d, e, f, g, h)
	}

	for j := 0; j < 256; j += 8 {
		if flag {
			a += ctx.randrsl[j]
			b += ctx.randrsl[j+1]
			c += ctx.randrsl[j+2]
			d += ctx.randrsl[j+3]
			e += ctx.randrsl[j+4]
			f += ctx.randrsl[j+5]
			g += ctx.randrsl[j+6]
			h += ctx.randrsl[j+7]
		}
		a, b, c, d, e, f, g, h = ctx.mix(a, b, c, d, e, f, g, h)
		ctx.mm[j] = a
		ctx.mm[j+1] = b
		ctx.mm[j+2] = c
		ctx.mm[j+3] = d
		ctx.mm[j+4] = e
		ctx.mm[j+5] = f
		ctx.mm[j+6] = g
		ctx.mm[j+7] = h
	}

	if flag {
		for j := 0; j < 256; j += 8 {
			a += ctx.mm[j]
			b += ctx.mm[j+1]
			c += ctx.mm[j+2]
			d += ctx.mm[j+3]
			e += ctx.mm[j+4]
			f += ctx.mm[j+5]
			g += ctx.mm[j+6]
			h += ctx.mm[j+7]
			a, b, c, d, e, f, g, h = ctx.mix(a, b, c, d, e, f, g, h)
			ctx.mm[j] = a
			ctx.mm[j+1] = b
			ctx.mm[j+2] = c
			ctx.mm[j+3] = d
			ctx.mm[j+4] = e
			ctx.mm[j+5] = f
			ctx.mm[j+6] = g
			ctx.mm[j+7] = h
		}
	}

	ctx.isaac64()
	ctx.randcnt = 256
}

func (ctx *Isaac64Ctx) mix(a, b, c, d, e, f, g, h uint64) (uint64, uint64, uint64, uint64, uint64, uint64, uint64, uint64) {
	a -= e
	f ^= h >> 9
	h += a
	b -= f
	g ^= a << 9
	a += b
	c -= g
	h ^= b >> 23
	b += c
	d -= h
	a ^= c << 15
	c += d
	e -= a
	b ^= d >> 14
	d += e
	f -= b
	c ^= e << 20
	e += f
	g -= c
	d ^= f >> 17
	f += g
	h -= d
	e ^= g << 14
	g += h
	return a, b, c, d, e, f, g, h
}

func (ctx *Isaac64Ctx) isaac64() {
	ctx.cc++
	ctx.bb += ctx.cc

	for j := 0; j < 256; j++ {
		x := ctx.mm[j]
		switch j % 4 {
		case 0:
			ctx.aa = ^(ctx.aa ^ (ctx.aa << 21))
		case 1:
			ctx.aa = ctx.aa ^ (ctx.aa >> 5)
		case 2:
			ctx.aa = ctx.aa ^ (ctx.aa << 12)
		case 3:
			ctx.aa = ctx.aa ^ (ctx.aa >> 33)
		}
		ctx.aa += ctx.mm[(j+128)%256]
		y := ctx.mm[(x>>3)%256] + ctx.aa + ctx.bb
		ctx.mm[j] = y
		ctx.bb = ctx.mm[(y>>11)%256] + x
		ctx.randrsl[j] = ctx.bb
	}
}

func (ctx *Isaac64Ctx) isaac64Random() uint64 {
	if ctx.randcnt == 0 {
		ctx.isaac64()
		ctx.randcnt = 256
	}
	ctx.randcnt--
	return ctx.randrsl[ctx.randcnt]
}
