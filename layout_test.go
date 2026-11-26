package main

import (
	"testing"
	"unsafe"
)

func TestPackedLayoutDoesNotExceedBudgetModel(t *testing.T) {
	if unsafe.Sizeof(QNode{}) > QNodeBytes {
		t.Fatalf("QNode size %d exceeds modeled %d", unsafe.Sizeof(QNode{}), QNodeBytes)
	}
	if unsafe.Sizeof(QTriangle{}) > QTriangleBytes {
		t.Fatalf("QTriangle size %d exceeds modeled %d", unsafe.Sizeof(QTriangle{}), QTriangleBytes)
	}
}
