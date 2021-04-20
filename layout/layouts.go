package layout

//import "math"
//
//type Size struct {
//	MinWidth, MaxWidth   float64
//	MinHeight, MaxHeight float64
//}
//
//func newSize() Size {
//	return Size{
//		MinWidth:  0.0,
//		MaxWidth:  math.Inf(1.0),
//		MinHeight: 0.0,
//		MaxHeight: math.Inf(1.0),
//	}
//}
//
//type Item interface {
//	Size() Size
//}
//
//type Layout struct {
//	items []Item
//}
//
//func (l *Layout) Size() Size {
//	size := newSize()
//	for _, item := range l.items {
//		isize := item.Size()
//		size.MinWidth = math.Max(size.MinWidth, isize.MinWidth)
//		size.MaxWidth = math.Min(size.MaxWidth, isize.MaxWidth)
//		size.MinHeight += isize.MinHeight
//		size.MaxHeight += isize.MaxHeight
//	}
//	return size
//}
