package pb

import "testing"

func Test_Width(t *testing.T) {
	count := 5000
	bar := New(count)
	width := 100
	bar.SetWidth(100).Callback = func(out string) {
		if len(out) != width {
			t.Errorf("Bar width expected {%d} was {%d}", len(out), width)
		}
	}
	bar.Start()
	bar.Set64(1)
	bar.Finish()
}

func Test_MultipleFinish(t *testing.T) {
	bar := New(5000)
	bar.Set64(7000)
	bar.Finish()
	bar.Finish()
}

func Test_Format(t *testing.T) {
	bar := New(5000).Format("[ooo]")
	bar.Set64(7000)
	bar.Finish()
	bar.Finish()
}
