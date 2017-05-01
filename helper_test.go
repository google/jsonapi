package jsonapi

import (
	"reflect"
	"testing"
	"time"
	"unsafe"
)

func TestHelper_BadPrimaryAnnotation(t *testing.T) {
	fields, err := extractFields(reflect.ValueOf(new(BadModel)))

	if fields != nil {
		t.Fatalf("Was expecting results to be nil")
	}

	if expected, actual := ErrBadJSONAPIStructTag, err; expected != actual {
		t.Fatalf("Was expecting error to be `%s`, got `%s`", expected, actual)
	}
}

func TestHelper_BadExtendedAnonymousField(t *testing.T) {
	fields, err := extractFields(reflect.ValueOf(new(WithBadExtendedAnonymousField)))

	if fields != nil {
		t.Fatalf("Was expecting results to be nil")
	}

	if expected, actual := ErrBadJSONAPIStructTag, err; expected != actual {
		t.Fatalf("Was expecting error to be `%s`, got `%s`", expected, actual)
	}
}

func TestHelper_returnsProperValue(t *testing.T) {
	comment := &Comment{}
	fields, err := extractFields(reflect.ValueOf(comment))

	if err != nil {
		t.Fatalf("Was expecting error to be nil, got `%s`", err)
	}

	if expected, actual := 4, len(fields); expected != actual {
		t.Fatalf("Was expecting fields to have `%d` items, got `%d`", expected, actual)
	}

	// Check Annotation value
	if expected, actual := "primary", fields[0].Annotation; expected != actual {
		t.Fatalf("Was expecting fields[0].Annotation to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := "client-id", fields[1].Annotation; expected != actual {
		t.Fatalf("Was expecting fields[1].Annotation to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := "attr", fields[2].Annotation; expected != actual {
		t.Fatalf("Was expecting fields[2].Annotation to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := "attr", fields[3].Annotation; expected != actual {
		t.Fatalf("Was expecting fields[3].Annotation to be `%s`, got `%s`", expected, actual)
	}

	// Check Args value
	if expected, actual := []string{"comments"}, fields[0].Args; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[0].Args to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := []string{}, fields[1].Args; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[1].Args to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := []string{"post_id"}, fields[2].Args; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[2].Args to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := []string{"body"}, fields[3].Args; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[3].Args to be `%s`, got `%s`", expected, actual)
	}

	// Check IsPtr
	if expected, actual := false, fields[0].IsPtr; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[0].IsPtr to be `%t`, got `%t`", expected, actual)
	}

	if expected, actual := false, fields[1].IsPtr; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[1].IsPtr to be `%t`, got `%t`", expected, actual)
	}

	if expected, actual := false, fields[2].IsPtr; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[2].IsPtr to be `%t`, got `%t`", expected, actual)
	}

	if expected, actual := false, fields[3].IsPtr; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[3].IsPtr to be `%t`, got `%t`", expected, actual)
	}

	// Check Value value
	if uintptr(unsafe.Pointer(&comment.ID)) != fields[0].Value.UnsafeAddr() {
		t.Fatalf("Was expecting fields[0].Value to point to comment.ID")
	}

	if uintptr(unsafe.Pointer(&comment.ClientID)) != fields[1].Value.UnsafeAddr() {
		t.Fatalf("Was expecting fields[1].Value to point to comment.ClientID")
	}

	if uintptr(unsafe.Pointer(&comment.PostID)) != fields[2].Value.UnsafeAddr() {
		t.Fatalf("Was expecting fields[2].Value to point to comment.PostID")
	}

	if uintptr(unsafe.Pointer(&comment.Body)) != fields[3].Value.UnsafeAddr() {
		t.Fatalf("Was expecting fields[3].Value to point to comment.Body")
	}

	// Check Kind value
	if expected, actual := reflect.Int, fields[0].Kind; expected != actual {
		t.Fatalf("Was expecting fields[0].Kind to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := reflect.String, fields[1].Kind; expected != actual {
		t.Fatalf("Was expecting fields[1].Kind to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := reflect.Int, fields[2].Kind; expected != actual {
		t.Fatalf("Was expecting fields[2].Kind to be `%s`, got `%s`", expected, actual)
	}

	if expected, actual := reflect.String, fields[3].Kind; expected != actual {
		t.Fatalf("Was expecting fields[3].Kind to be `%s`, got `%s`", expected, actual)
	}

}

func TestHelper_ignoreFieldWithoutAnnotation(t *testing.T) {
	book := &Book{
		ID:          0,
		Author:      "aren55555",
		PublishedAt: time.Now().AddDate(0, -1, 0),
	}
	fields, err := extractFields(reflect.ValueOf(book))
	if err != nil {
		t.Fatalf("Was expecting error to be nil, got `%s`", err)
	}

	if expected, actual := 7, len(fields); expected != actual {
		t.Fatalf("Was expecting fields to have `%d` items, got `%d`", expected, actual)
	}
}

func TestHelper_WithExtendedAnonymousField(t *testing.T) {
	model := &WithExtendedAnonymousField{}
	fields, err := extractFields(reflect.ValueOf(model))
	if err != nil {
		t.Fatalf("Was expecting error to be nil, got `%s`", err)
	}

	if expected, actual := 2, len(fields); expected != actual {
		t.Fatalf("Was expecting fields to have `%d` items, got `%d`", expected, actual)
	}

	if uintptr(unsafe.Pointer(&model.CommonField)) != fields[0].Value.UnsafeAddr() {
		t.Fatalf("Was expecting fields[0].Value to point to comment.CommonField")
	}

	if uintptr(unsafe.Pointer(&model.ID)) != fields[1].Value.UnsafeAddr() {
		t.Fatalf("Was expecting fields[1].Value to point to comment.ID")
	}
}

func TestHelper_WithPointer(t *testing.T) {
	model := &WithPointer{}
	fields, err := extractFields(reflect.ValueOf(model))
	if err != nil {
		t.Fatalf("Was expecting error to be nil, got `%s`", err)
	}

	if expected, actual := 5, len(fields); expected != actual {
		t.Fatalf("Was expecting fields to have `%d` items, got `%d`", expected, actual)
	}

	if expected, actual := true, fields[0].IsPtr; !reflect.DeepEqual(expected, actual) {
		t.Fatalf("Was expecting fields[0].IsPtr to be `%t`, got `%t`", expected, actual)
	}

	if uintptr(unsafe.Pointer(&model.ID)) != fields[0].Value.UnsafeAddr() {
		t.Fatalf("Was expecting fields[0].Value to point to comment.ID")
	}
}
