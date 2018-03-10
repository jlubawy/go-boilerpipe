package boilerpipe

import (
	"testing"
)

func TestLabelStack(t *testing.T) {
	labelStack := NewLabelStack()

	labelStack.Push(LabelHeading, LabelHeading1, LabelHeading2)

	if l := labelStack.Len(); l != 3 {
		t.Fatalf("expected length of 3 but got %d", l)
	}

	label, ok := labelStack.Pop()
	if !ok {
		t.Fatal("expected ok")
	}

	if label != LabelHeading2 {
		t.Fatalf("expected LabelHeading2 but got %s", label)
	}

	labels := labelStack.PopAll()
	if l := len(labels); l != 2 {
		t.Fatalf("expected 2 labels but got %d", l)
	}

	if labels[0] != LabelHeading1 {
		t.Fatalf("expected LabelHeading1 but got %s", label)
	}
	if labels[1] != LabelHeading {
		t.Fatalf("expected LabelHeading but got %s", label)
	}

	if l := labelStack.Len(); l != 0 {
		t.Fatalf("expected 0 labels but got %d", l)
	}
}

//func NewLabelStack() *LabelStack {
//    return &LabelStack{
//        labels: make([]Label, 0),
//    }
//}
//
//func (x *LabelStack) Pop() (label Label, ok bool) {
//    if len(x.labels) == 0 {
//        return
//    }
//    label = x.labels[len(x.labels)-1]
//    ok = true
//    x.labels = x.labels[:len(x.labels)-1]
//    return
//}
//
//func (x *LabelStack) PopAll() (labels []Label) {
//    labels = x.labels
//    x.labels = nil
//    x.labels = make([]Label, 0)
//    return
//}
//
//func (x *LabelStack) Push(labels ...Label) {
//    x.labels = append(x.labels, labels...)
//}
