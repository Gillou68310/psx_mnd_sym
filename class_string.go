// Code generated by "stringer -linecomment -type Class"; DO NOT EDIT.

package sym

import "strconv"

const (
	_Class_name_0 = "EXTSTAT"
	_Class_name_1 = "MOS"
	_Class_name_2 = "STRTAGMOUUNTAGTPDEF"
	_Class_name_3 = "ENTAGMOE"
	_Class_name_4 = "FIELD"
	_Class_name_5 = "EOS"
)

var (
	_Class_index_0 = [...]uint8{0, 3, 7}
	_Class_index_2 = [...]uint8{0, 6, 9, 14, 19}
	_Class_index_3 = [...]uint8{0, 5, 8}
)

func (i Class) String() string {
	switch {
	case 2 <= i && i <= 3:
		i -= 2
		return _Class_name_0[_Class_index_0[i]:_Class_index_0[i+1]]
	case i == 8:
		return _Class_name_1
	case 10 <= i && i <= 13:
		i -= 10
		return _Class_name_2[_Class_index_2[i]:_Class_index_2[i+1]]
	case 15 <= i && i <= 16:
		i -= 15
		return _Class_name_3[_Class_index_3[i]:_Class_index_3[i+1]]
	case i == 18:
		return _Class_name_4
	case i == 102:
		return _Class_name_5
	default:
		return "Class(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
