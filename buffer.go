package main

import "bytes"

type buffer struct {
	*bytes.Buffer
}

func newBuffer() *buffer {
	return &buffer{&bytes.Buffer{}}
}

// Close ...
func (b *buffer) Close() error {
	return nil
}

// WriteLine ...
func (b *buffer) WriteLine(bs []byte) (int, error) {
	wrtn := 0

	if len(bs) > 0 {
		n, err := b.Write(bs)
		wrtn += n
		if err != nil {
			return wrtn, err
		}
	}

	_ = b.WriteByte('\n')
	wrtn++

	return wrtn, nil
}
