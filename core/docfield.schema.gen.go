package core

import (
	"io"
	"time"
	"unsafe"
)

var (
	_ = unsafe.Sizeof(0)
	_ = io.ReadFull
	_ = time.Now()
)

type DocField struct {
	DocTokenLen float32
	GeoHash     string
	Lat         float64
	Lng         float64
}

func (d *DocField) Size() (s uint64) {

	{
		l := uint64(len(d.GeoHash))

		{

			t := l
			for t >= 0x80 {
				t >>= 7
				s++
			}
			s++

		}
		s += l
	}
	s += 20
	return
}
func (d *DocField) Marshal(buf []byte) ([]byte, error) {
	size := d.Size()
	{
		if uint64(cap(buf)) >= size {
			buf = buf[:size]
		} else {
			buf = make([]byte, size)
		}
	}
	i := uint64(0)

	{

		v := *(*uint32)(unsafe.Pointer(&(d.DocTokenLen)))

		buf[0+0] = byte(v >> 0)

		buf[1+0] = byte(v >> 8)

		buf[2+0] = byte(v >> 16)

		buf[3+0] = byte(v >> 24)

	}
	{
		l := uint64(len(d.GeoHash))

		{

			t := uint64(l)

			for t >= 0x80 {
				buf[i+4] = byte(t) | 0x80
				t >>= 7
				i++
			}
			buf[i+4] = byte(t)
			i++

		}
		copy(buf[i+4:], d.GeoHash)
		i += l
	}
	{

		v := *(*uint64)(unsafe.Pointer(&(d.Lat)))

		buf[i+0+4] = byte(v >> 0)

		buf[i+1+4] = byte(v >> 8)

		buf[i+2+4] = byte(v >> 16)

		buf[i+3+4] = byte(v >> 24)

		buf[i+4+4] = byte(v >> 32)

		buf[i+5+4] = byte(v >> 40)

		buf[i+6+4] = byte(v >> 48)

		buf[i+7+4] = byte(v >> 56)

	}
	{

		v := *(*uint64)(unsafe.Pointer(&(d.Lng)))

		buf[i+0+12] = byte(v >> 0)

		buf[i+1+12] = byte(v >> 8)

		buf[i+2+12] = byte(v >> 16)

		buf[i+3+12] = byte(v >> 24)

		buf[i+4+12] = byte(v >> 32)

		buf[i+5+12] = byte(v >> 40)

		buf[i+6+12] = byte(v >> 48)

		buf[i+7+12] = byte(v >> 56)

	}
	return buf[:i+20], nil
}

func (d *DocField) Unmarshal(buf []byte) (uint64, error) {
	i := uint64(0)

	{

		v := 0 | (uint32(buf[i+0+0]) << 0) | (uint32(buf[i+1+0]) << 8) | (uint32(buf[i+2+0]) << 16) | (uint32(buf[i+3+0]) << 24)
		d.DocTokenLen = *(*float32)(unsafe.Pointer(&v))

	}
	{
		l := uint64(0)

		{

			bs := uint8(7)
			t := uint64(buf[i+4] & 0x7F)
			for buf[i+4]&0x80 == 0x80 {
				i++
				t |= uint64(buf[i+4]&0x7F) << bs
				bs += 7
			}
			i++

			l = t

		}
		d.GeoHash = string(buf[i+4 : i+4+l])
		i += l
	}
	{

		v := 0 | (uint64(buf[i+0+4]) << 0) | (uint64(buf[i+1+4]) << 8) | (uint64(buf[i+2+4]) << 16) | (uint64(buf[i+3+4]) << 24) | (uint64(buf[i+4+4]) << 32) | (uint64(buf[i+5+4]) << 40) | (uint64(buf[i+6+4]) << 48) | (uint64(buf[i+7+4]) << 56)
		d.Lat = *(*float64)(unsafe.Pointer(&v))

	}
	{

		v := 0 | (uint64(buf[i+0+12]) << 0) | (uint64(buf[i+1+12]) << 8) | (uint64(buf[i+2+12]) << 16) | (uint64(buf[i+3+12]) << 24) | (uint64(buf[i+4+12]) << 32) | (uint64(buf[i+5+12]) << 40) | (uint64(buf[i+6+12]) << 48) | (uint64(buf[i+7+12]) << 56)
		d.Lng = *(*float64)(unsafe.Pointer(&v))

	}
	return i + 20, nil
}
