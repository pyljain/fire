package main

import "fmt"

type HeadersArray []string

func (h *HeadersArray) Set(val string) error {
	*h = append(*h, val)
	return nil
}

func (h *HeadersArray) String() string {
	return fmt.Sprintf("%+v", *h)
}
