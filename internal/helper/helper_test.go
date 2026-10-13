package helper

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestFormatAAAARecordSet(t *testing.T) {
	testCases := []struct {
		name string
		old  []interface{}
		new  []interface{}
		want bool
	}{
		{
			name: "compressed and expanded notation",
			old:  []interface{}{"2001:db8::10"},
			new:  []interface{}{"2001:db8:0:0:0:0:0:10"},
			want: true,
		},
		{
			name: "equivalent values in different order",
			old:  []interface{}{"2001:db8::10", "2001:db8::20"},
			new:  []interface{}{"2001:db8:0:0:0:0:0:20", "2001:db8:0:0:0:0:0:10"},
			want: true,
		},
		{
			name: "different address",
			old:  []interface{}{"2001:db8::10"},
			new:  []interface{}{"2001:db8::11"},
			want: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			oldSet := schema.NewSet(schema.HashString, testCase.old)
			newSet := schema.NewSet(schema.HashString, testCase.new)

			if got := formatAAAARecordSet(oldSet).Equal(formatAAAARecordSet(newSet)); got != testCase.want {
				t.Errorf("formatAAAARecordSet equality = %v, want %v", got, testCase.want)
			}
		})
	}
}
