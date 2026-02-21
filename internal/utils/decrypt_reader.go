package utils

import (
	"encoding/binary"
	"io"

	"wx_channel/pkg/util"
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
// 用于生成解密密钥流
type Isaac64Ctx struct {
	randrsl [256]uint64
	randcnt uint64
	mm      [256]uint64
	aa      uint64
	bb      uint64
	cc      uint64
}

// NewDecryptReader 创建一个新的解密读取器
// reader: 底层数据源
// key: ISAAC64 种子（解密密钥）
// offset: 起始偏移（用于 Range 请求）
// limit: 加密区域大小（通常为 131072 字节 = 128KB）
func NewDecryptReader(reader io.Reader, key uint64, offset uint64, limit uint64) *DecryptReader {
	ctx := createIsaac64Context(key)
	dr := &DecryptReader{
		reader:   reader,
		ctx:      ctx,
		limit:    limit,
		consumed: 0,
		ksPos:    8, // 初始化为 8，表示需要生成新的密钥块
	}

	if limit > 0 {
		// 将 consumed 对齐到文件偏移，超出加密区则设置为加密区末尾
		if offset >= limit {
			dr.consumed = limit
		} else {
			dr.consumed = offset
			// 跳过完整的 8 字节密钥块
			skipBlocks := offset / 8
			for i := uint64(0); i < skipBlocks; i++ {
				_ = dr.ctx.isaac64Random()
			}
			// 生成当前块并设置起始位置
			randNumber := dr.ctx.isaac64Random()
			binary.BigEndian.PutUint64(dr.ks[:], randNumber)
			dr.ksPos = int(offset % 8)
		}
	}
	return dr
}

// Read 实现 io.Reader 接口
// 从底层读取器读取数据，并对加密区域进行 XOR 解密
func (dr *DecryptReader) Read(p []byte) (int, error) {
	n, err := dr.reader.Read(p)
	if n <= 0 {
		return n, err
	}

	// 如果没有加密区域限制或已超出加密区域，直接返回
	if dr.limit == 0 || dr.consumed >= dr.limit {
		return n, err
	}

	// 计算需要解密的字节数
	toDecrypt := uint64(n)
	remaining := dr.limit - dr.consumed
	if toDecrypt > remaining {
		toDecrypt = remaining
	}

	// 逐字节异或解密，维护密钥流位置
	for i := uint64(0); i < toDecrypt; i++ {
		if dr.ksPos >= 8 {
			// 需要生成新的密钥块
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

// createIsaac64Context 创建并初始化 ISAAC64 上下文
func createIsaac64Context(seed uint64) *Isaac64Ctx {
	isaac := util.NewIsaac64(seed)
	return &Isaac64Ctx{
		randrsl: isaac.GetRandrsl(),
		randcnt: isaac.GetRandcnt(),
		mm:      isaac.GetMm(),
		aa:      isaac.GetAa(),
		bb:      isaac.GetBb(),
		cc:      isaac.GetCc(),
	}
}

// isaac64Random 生成下一个随机数
func (ctx *Isaac64Ctx) isaac64Random() uint64 {
	if ctx.randcnt == 0 {
		ctx.isaac64()
		ctx.randcnt = 256
	}
	ctx.randcnt--
	return ctx.randrsl[ctx.randcnt]
}

// isaac64 执行 ISAAC64 算法的核心迭代
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
