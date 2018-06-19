package mish

// import "testing"

// func TestSnapCursorToBlock(t *testing.T) {
// 	testCases := []struct {
// 		before   Cursor
// 		blocks   []int
// 		expected Cursor
// 	}{
// 		{Cursor{Block: 1, Line: 10}, []int{10, 10, 50}, Cursor{Block: 1, Line: 10}},
// 		{Cursor{Block: 1, Line: 40}, []int{10, 10, 50}, Cursor{Block: 2, Line: 30}},
// 		{Cursor{Block: 1, Line: 70}, []int{10, 10, 50}, Cursor{Block: 2, Line: 50}},
// 		{Cursor{Block: 2, Line: -15}, []int{10, 10, 50}, Cursor{Block: 0, Line: 5}},
// 		{Cursor{Block: 1, Line: -30}, []int{10, 10, 50}, Cursor{Block: 0, Line: 0}},
// 	}
// 	for i, tc := range testCases {
// 		actual := snapCursorToBlock(tc.before, tc.blocks)
// 		if actual != tc.expected {
// 			t.Fatalf("Test %d: Expected %+v, got %+v", i, tc.expected, actual)
// 		}
// 	}
// }
