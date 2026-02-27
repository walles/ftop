package ftop

func (u *Ui) renderProcessInfoPane(y0, y1 int) {
	if u.pickedProcess == nil {
		panic("no process picked")
	}

	width, _ := u.screen.Size()
	x1 := width - 1

	defer renderFrame(u.screen, u.theme, 0, y0, x1, y1, "Process Info")

	// FIXME: Render the process hierarchy
}
