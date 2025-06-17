package testutil

import (
	"testing"
)

func TestIntPtr(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"positive value", 42},
		{"negative value", -42},
		{"zero value", 0},
		{"large value", 2147483647},   // max int32
		{"small value", -2147483648},  // min int32
		{"one", 1},
		{"minus one", -1},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr := IntPtr(tt.value)
			
			// Verify pointer is not nil
			if ptr == nil {
				t.Fatal("IntPtr should not return nil")
			}
			
			// Verify dereferenced value equals original
			if *ptr != tt.value {
				t.Errorf("Expected *IntPtr(%d) = %d, got %d", tt.value, tt.value, *ptr)
			}
			
			// Verify it's actually a pointer to int
			var _ *int = ptr
		})
	}
}

func TestInt64Ptr(t *testing.T) {
	tests := []struct {
		name  string
		value int64
	}{
		{"positive value", int64(42)},
		{"negative value", int64(-42)},
		{"zero value", int64(0)},
		{"large value", int64(9223372036854775807)},   // max int64
		{"small value", int64(-9223372036854775808)},  // min int64
		{"one", int64(1)},
		{"minus one", int64(-1)},
		{"int32 max as int64", int64(2147483647)},
		{"int32 min as int64", int64(-2147483648)},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptr := Int64Ptr(tt.value)
			
			// Verify pointer is not nil
			if ptr == nil {
				t.Fatal("Int64Ptr should not return nil")
			}
			
			// Verify dereferenced value equals original
			if *ptr != tt.value {
				t.Errorf("Expected *Int64Ptr(%d) = %d, got %d", tt.value, tt.value, *ptr)
			}
			
			// Verify it's actually a pointer to int64
			var _ *int64 = ptr
		})
	}
}

func TestIntPtr_UniquePointers(t *testing.T) {
	// Test that multiple calls with the same value return different pointers
	value := 42
	ptr1 := IntPtr(value)
	ptr2 := IntPtr(value)
	
	// Pointers should be different (different memory addresses)
	if ptr1 == ptr2 {
		t.Error("IntPtr should return different pointers for each call")
	}
	
	// But values should be the same
	if *ptr1 != *ptr2 {
		t.Errorf("Values should be equal: *ptr1=%d, *ptr2=%d", *ptr1, *ptr2)
	}
	
	// Both should equal the original value
	if *ptr1 != value || *ptr2 != value {
		t.Errorf("Both pointers should point to %d, got *ptr1=%d, *ptr2=%d", value, *ptr1, *ptr2)
	}
}

func TestInt64Ptr_UniquePointers(t *testing.T) {
	// Test that multiple calls with the same value return different pointers
	value := int64(42)
	ptr1 := Int64Ptr(value)
	ptr2 := Int64Ptr(value)
	
	// Pointers should be different (different memory addresses)
	if ptr1 == ptr2 {
		t.Error("Int64Ptr should return different pointers for each call")
	}
	
	// But values should be the same
	if *ptr1 != *ptr2 {
		t.Errorf("Values should be equal: *ptr1=%d, *ptr2=%d", *ptr1, *ptr2)
	}
	
	// Both should equal the original value
	if *ptr1 != value || *ptr2 != value {
		t.Errorf("Both pointers should point to %d, got *ptr1=%d, *ptr2=%d", value, *ptr1, *ptr2)
	}
}

func TestIntPtr_Modification(t *testing.T) {
	// Test that modifying the pointed value works correctly
	originalValue := 42
	ptr := IntPtr(originalValue)
	
	// Verify initial value
	if *ptr != originalValue {
		t.Errorf("Expected initial value %d, got %d", originalValue, *ptr)
	}
	
	// Modify the value through the pointer
	newValue := 84
	*ptr = newValue
	
	// Verify the value was changed
	if *ptr != newValue {
		t.Errorf("Expected modified value %d, got %d", newValue, *ptr)
	}
	
	// Original variable should be unchanged (since we have a copy)
	if originalValue != 42 {
		t.Errorf("Original value should be unchanged, got %d", originalValue)
	}
}

func TestInt64Ptr_Modification(t *testing.T) {
	// Test that modifying the pointed value works correctly
	originalValue := int64(42)
	ptr := Int64Ptr(originalValue)
	
	// Verify initial value
	if *ptr != originalValue {
		t.Errorf("Expected initial value %d, got %d", originalValue, *ptr)
	}
	
	// Modify the value through the pointer
	newValue := int64(84)
	*ptr = newValue
	
	// Verify the value was changed
	if *ptr != newValue {
		t.Errorf("Expected modified value %d, got %d", newValue, *ptr)
	}
	
	// Original variable should be unchanged (since we have a copy)
	if originalValue != 42 {
		t.Errorf("Original value should be unchanged, got %d", originalValue)
	}
}

func TestPointerHelpers_Integration(t *testing.T) {
	// Test using both pointer helpers together
	intVal := 42
	int64Val := int64(84)
	
	intPtr := IntPtr(intVal)
	int64Ptr := Int64Ptr(int64Val)
	
	// Verify both pointers work
	if *intPtr != intVal {
		t.Errorf("IntPtr failed: expected %d, got %d", intVal, *intPtr)
	}
	
	if *int64Ptr != int64Val {
		t.Errorf("Int64Ptr failed: expected %d, got %d", int64Val, *int64Ptr)
	}
	
	// Test that we can convert between types through pointers
	*intPtr = int(*int64Ptr)
	if *intPtr != 84 {
		t.Errorf("Type conversion failed: expected 84, got %d", *intPtr)
	}
}

func TestPointerHelpers_NilComparison(t *testing.T) {
	// Test that the returned pointers are not nil
	intPtr := IntPtr(0)
	int64Ptr := Int64Ptr(0)
	
	if intPtr == nil {
		t.Error("IntPtr(0) should not return nil")
	}
	
	if int64Ptr == nil {
		t.Error("Int64Ptr(0) should not return nil")
	}
	
	// Even for zero values, we should get valid pointers
	if *intPtr != 0 {
		t.Errorf("IntPtr(0) should point to 0, got %d", *intPtr)
	}
	
	if *int64Ptr != 0 {
		t.Errorf("Int64Ptr(0) should point to 0, got %d", *int64Ptr)
	}
}

func TestPointerHelpers_TypeSafety(t *testing.T) {
	// Test that the functions return the correct types
	intPtr := IntPtr(42)
	int64Ptr := Int64Ptr(42)
	
	// These should compile without issues
	var _ *int = intPtr
	var _ *int64 = int64Ptr
	
	// These should not be assignable to each other
	// (This is a compile-time check, but we can verify the types)
	if intPtr == nil {
		t.Error("intPtr should not be nil")
	}
	if int64Ptr == nil {
		t.Error("int64Ptr should not be nil")
	}
}

func TestPointerHelpers_EdgeCases(t *testing.T) {
	// Test with extreme values
	tests := []struct {
		name     string
		intVal   int
		int64Val int64
	}{
		{"max values", int(2147483647), int64(9223372036854775807)},
		{"min values", int(-2147483648), int64(-9223372036854775808)},
		{"zero values", 0, 0},
		{"one values", 1, 1},
		{"minus one values", -1, -1},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intPtr := IntPtr(tt.intVal)
			int64Ptr := Int64Ptr(tt.int64Val)
			
			if *intPtr != tt.intVal {
				t.Errorf("IntPtr failed for %s: expected %d, got %d", tt.name, tt.intVal, *intPtr)
			}
			
			if *int64Ptr != tt.int64Val {
				t.Errorf("Int64Ptr failed for %s: expected %d, got %d", tt.name, tt.int64Val, *int64Ptr)
			}
		})
	}
}

// Benchmark the pointer helper functions
func BenchmarkIntPtr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ptr := IntPtr(42)
		if ptr == nil {
			b.Fatal("IntPtr returned nil")
		}
	}
}

func BenchmarkInt64Ptr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ptr := Int64Ptr(42)
		if ptr == nil {
			b.Fatal("Int64Ptr returned nil")
		}
	}
}

func BenchmarkPointerDereference(b *testing.B) {
	intPtr := IntPtr(42)
	int64Ptr := Int64Ptr(42)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = *intPtr
		_ = *int64Ptr
	}
}