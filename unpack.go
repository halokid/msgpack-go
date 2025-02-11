package msgpack

import (
	"io"
	"reflect"
	"unsafe"
	//"bytes"
	//"strings"
	"log"
)

type (
	Bytes1 [1]byte
	Bytes2 [2]byte
	Bytes4 [4]byte
	Bytes8 [8]byte
)

const (
	NEGFIXNUM     = 0xe0
	FIXMAPMAX     = 0x8f
	FIXARRAYMAX   = 0x9f
	FIXRAWMAX     = 0xbf
	FIRSTBYTEMASK = 0xf
)

func readByte(reader io.Reader) (v uint8, err error) {
	var data Bytes1
	_, e := reader.Read(data[0:])
	if e != nil {
		return 0, e
	}
	return data[0], nil
}

func readUint16(reader io.Reader) (v uint16, n int, err error) {
	var data Bytes2
	n, e := reader.Read(data[0:])
	if e != nil {
		return 0, n, e
	}
	return (uint16(data[0]) << 8) | uint16(data[1]), n, nil
}

func readUint32(reader io.Reader) (v uint32, n int, err error) {
	var data Bytes4
	n, e := reader.Read(data[0:])
	if e != nil {
		return 0, n, e
	}
	return (uint32(data[0]) << 24) | (uint32(data[1]) << 16) | (uint32(data[2]) << 8) | uint32(data[3]), n, nil
}

func readUint64(reader io.Reader) (v uint64, n int, err error) {
	var data Bytes8
	n, e := reader.Read(data[0:])
	if e != nil {
		return 0, n, e
	}
	return (uint64(data[0]) << 56) | (uint64(data[1]) << 48) | (uint64(data[2]) << 40) | (uint64(data[3]) << 32) | (uint64(data[4]) << 24) | (uint64(data[5]) << 16) | (uint64(data[6]) << 8) | uint64(data[7]), n, nil
}

func readInt16(reader io.Reader) (v int16, n int, err error) {
	var data Bytes2
	n, e := reader.Read(data[0:])
	if e != nil {
		return 0, n, e
	}
	return (int16(data[0]) << 8) | int16(data[1]), n, nil
}

func readInt32(reader io.Reader) (v int32, n int, err error) {
	var data Bytes4
	n, e := reader.Read(data[0:])
	if e != nil {
		return 0, n, e
	}
	return (int32(data[0]) << 24) | (int32(data[1]) << 16) | (int32(data[2]) << 8) | int32(data[3]), n, nil
}

func readInt64(reader io.Reader) (v int64, n int, err error) {
	var data Bytes8
	n, e := reader.Read(data[0:])
	if e != nil {
		return 0, n, e
	}
	return (int64(data[0]) << 56) | (int64(data[1]) << 48) | (int64(data[2]) << 40) | (int64(data[3]) << 32) | (int64(data[4]) << 24) | (int64(data[5]) << 16) | (int64(data[6]) << 8) | int64(data[7]), n, nil
}

func unpackArray(reader io.Reader, nelems uint) (v reflect.Value, n int, err error) {
	var i uint
	var nbytesread int
	retval := make([]interface{}, nelems)

	for i = 0; i < nelems; i++ {
		v, n, err = Unpack(reader)
		nbytesread += n
		if err != nil {
			return reflect.Value{}, nbytesread, err
		}
		retval[i] = v.Interface()
	}
	return reflect.ValueOf(retval), nbytesread, nil
}

func unpackArrayReflected(reader io.Reader, nelems uint) (v reflect.Value, n int, err error) {
	var i uint
	var nbytesread int
	retval := make([]reflect.Value, nelems)

	for i = 0; i < nelems; i++ {
		v, n, err = UnpackReflected(reader)
		nbytesread += n
		if err != nil {
			return reflect.Value{}, nbytesread, err
		}
		retval[i] = v
	}
	return reflect.ValueOf(retval), nbytesread, nil
}

func unpackMap(reader io.Reader, nelems uint) (v reflect.Value, n int, err error) {
	var i uint
	var nbytesread int
	var k reflect.Value
	retval := make(map[interface{}]interface{})

	for i = 0; i < nelems; i++ {
		k, n, err = Unpack(reader)
		nbytesread += n
		if err != nil {
			return reflect.Value{}, nbytesread, err
		}
		v, n, err = Unpack(reader)
		nbytesread += n
		if err != nil {
			return reflect.Value{}, nbytesread, err
		}
		ktyp := k.Type()
		if ktyp.Kind() == reflect.Slice && ktyp.Elem().Kind() == reflect.Uint8 {
			retval[string(k.Interface().([]uint8))] = v.Interface()
		} else {
			retval[k.Interface()] = v.Interface()
		}
	}
	return reflect.ValueOf(retval), nbytesread, nil
}

func unpackMapReflected(reader io.Reader, nelems uint) (v reflect.Value, n int, err error) {
	var i uint
	var nbytesread int
	var k reflect.Value
	retval := make(map[interface{}]reflect.Value)

	for i = 0; i < nelems; i++ {
		k, n, err = UnpackReflected(reader)
		nbytesread += n
		if err != nil {
			return reflect.Value{}, nbytesread, err
		}
		v, n, err = UnpackReflected(reader)
		nbytesread += n
		if err != nil {
			return reflect.Value{}, nbytesread, err
		}
		retval[k] = v
	}
	return reflect.ValueOf(retval), nbytesread, nil
}

// Get the four lowest bits
func lownibble(u8 uint8) uint {
	return uint(u8 & 0xf)
}

// Get the five lowest bits
func lowfive(u8 uint8) uint {
	return uint(u8 & 0x1f)
}

func unpack(reader io.Reader, reflected bool) (v reflect.Value, n int, err error) {
	var retval reflect.Value
	var nbytesread int

	c, e := readByte(reader)
	//log.Printf("debug c ------ %+v, err ------- %+v", c, e)
	if e != nil {
		return reflect.Value{}, 0, e
	}
	nbytesread++
	if c < FIXMAP || c >= NEGFIXNUM {
		retval = reflect.ValueOf(int8(c))
	} else if c >= FIXMAP && c <= FIXMAPMAX {
		//log.Printf("---------------- debug 4 ---------------")
		if reflected {
			retval, n, e = unpackMapReflected(reader, lownibble(c))
		} else {
			retval, n, e = unpackMap(reader, lownibble(c))
		}
		nbytesread += n
		if e != nil {
			return reflect.Value{}, nbytesread, e
		}
		nbytesread += n
	} else if c >= FIXARRAY && c <= FIXARRAYMAX {
		//log.Printf("---------------- debug 3 ---------------")
		if reflected {
			retval, n, e = unpackArrayReflected(reader, lownibble(c))
		} else {
			retval, n, e = unpackArray(reader, lownibble(c))
		}
		nbytesread += n
		if e != nil {
			return reflect.Value{}, nbytesread, e
		}
		nbytesread += n
	} else if c >= FIXRAW && c <= FIXRAWMAX {
		//log.Printf("---------------- debug 2 ---------------")
		data := make([]byte, lowfive(c))
		n, e := reader.Read(data)
		nbytesread += n
		if e != nil {
			return reflect.Value{}, nbytesread, e
		}
		retval = reflect.ValueOf(data)
	} else {
		//log.Printf("---------------- debug 1 ---------------")
		switch c {
		case NIL:
			retval = reflect.ValueOf(nil)
		case FALSE:
			retval = reflect.ValueOf(false)
		case TRUE:
			retval = reflect.ValueOf(true)
		case FLOAT:
			data, n, e := readUint32(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(*(*float32)(unsafe.Pointer(&data)))
		case DOUBLE:
			data, n, e := readUint64(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(*(*float64)(unsafe.Pointer(&data)))
		case UINT8:
			data, e := readByte(reader)
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(uint8(data))
			nbytesread++
		case UINT16:
			data, n, e := readUint16(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case UINT32:
			data, n, e := readUint32(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case UINT64:
			data, n, e := readUint64(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case INT8:
			data, e := readByte(reader)
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(int8(data))
			nbytesread++
		case INT16:
			data, n, e := readInt16(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case INT32:
			data, n, e := readInt32(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case INT64:
			data, n, e := readInt64(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case RAW16:
			nbytestoread, n, e := readUint16(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			data := make([]byte, nbytestoread)
			n, e = reader.Read(data)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case RAW32:
			nbytestoread, n, e := readUint32(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			data := make(Bytes, nbytestoread)
			n, e = reader.Read(data)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			retval = reflect.ValueOf(data)
		case ARRAY16:
			nelemstoread, n, e := readUint16(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			if reflected {
				retval, n, e = unpackArrayReflected(reader, uint(nelemstoread))
			} else {
				retval, n, e = unpackArray(reader, uint(nelemstoread))
			}
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
		case ARRAY32:
			nelemstoread, n, e := readUint32(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			if reflected {
				retval, n, e = unpackArrayReflected(reader, uint(nelemstoread))
			} else {
				retval, n, e = unpackArray(reader, uint(nelemstoread))
			}
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
		case MAP16:
			nelemstoread, n, e := readUint16(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			if reflected {
				retval, n, e = unpackMapReflected(reader, uint(nelemstoread))
			} else {
				retval, n, e = unpackMap(reader, uint(nelemstoread))
			}
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
		case MAP32:
			nelemstoread, n, e := readUint32(reader)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			if reflected {
				retval, n, e = unpackMapReflected(reader, uint(nelemstoread))
			} else {
				retval, n, e = unpackMap(reader, uint(nelemstoread))
			}
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
		default:
			//panic("unsupported code: " + strconv.Itoa(int(c)))
			
			//log.Println("unsupported code: " + strconv.Itoa(int(c)))
			//log.Printf("---------------- debug 99 ---------------")
			//readerLen, err := reader.Read()
			//log.Printf("reader --------------- %+v", readerLen)
			//buf := new(bytes.Buffer)
			//buf.ReadFrom(reader)			// todo: this will block the reader
			//dataLen := buf.Len()
			//log.Printf("buf --------------- %+v", buf)
			
			//buf := &bytes.Buffer{}
			//nRead, err := io.Copy(buf, reader)
			//log.Println("nRead ------------", nRead)
			//if err != nil {
			//	log.Println(err)
			//}
		
			// todo: 匹配一些解码不了，但是又可以读取的请况, 这种方式是先设置大小，最大设置为2M, 再去读取
			log.Println("---匹配到一些msgpack解码不了，但是又可以正确读取的情况, dataLen设置为2M---")
			log.Println("---现在网关GW有一些接口数据比较大，接口返回乱码的问题会不会是因为这里？但是乱码时却没有跑这段逻辑---")
			dataLen := 1024 * 1024 * 2
			//log.Printf("dataLen ------------ %+v", dataLen)
			data := make([]byte, dataLen) 
			n, e := reader.Read(data)
			//log.Println("e -------------- ", e)
			nbytesread += n
			if e != nil {
				return reflect.Value{}, nbytesread, e
			}
			//dataFixBlank := bytes.Trim(data[1:], " ")
			//log.Println("data len --------------", len(data))
			//dataFixBlank := strings.TrimRight(string(data[1:]), " ")
			dataFixBlank := data[1:]		// todo: 这种请况前面会多一个符号，要去掉这个符号
			//log.Println("dataFixBlank --------------", dataFixBlank[200])
			//log.Println("dataFixBlank len --------------", len(dataFixBlank))
			//for i, j := range dataFixBlank {
				//log.Printf("dataFixBlank item ----------- %+v, %+v, %+v\n", i, reflect.TypeOf(i), j)
			//}
			cutIdx := 0
			for i, v := range dataFixBlank {
				if v == 0 {
					cutIdx = i
					break
				}
			}
			//log.Println("cutIdx -----------", cutIdx)
			dataFix := dataFixBlank[:cutIdx]

			//retval = reflect.ValueOf(data)
			//retval = reflect.ValueOf(dataFixBlank)
			retval = reflect.ValueOf(dataFix)

			// -------------------------
			//var buf = make([]byte, 1024)
			//nRead, e := io.ReadFull(reader, buf)
			//log.Println("e -------------- ", e)
			//if nRead > 0 && nRead <= 1023 {
			//	data := (buf[:nRead])
			//	retval = reflect.ValueOf(data)
			//}
		}
		
	}
	return retval, nbytesread, nil
}

// Reads a value from the reader, unpack and returns it.
func Unpack(reader io.Reader) (v reflect.Value, n int, err error) {
	return unpack(reader, false)
}

// Reads unpack a value from the reader, unpack and returns it.  When the
// value is an array or map, leaves the elements wrapped by corresponding
// wrapper objects defined in reflect package.
func UnpackReflected(reader io.Reader) (v reflect.Value, n int, err error) {
	return unpack(reader, true)
}
