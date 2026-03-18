package tui

// isVerticalLayout returns true when terminal is too narrow for side-by-side.
func (a *App) isVerticalLayout() bool {
	return a.width < 80
}

// sideWidth calculates the left panel width, shrinking for narrow terminals.
func (a *App) sideWidth() int {
	if a.isVerticalLayout() {
		return a.width
	}
	sideW := a.cfg.GUI.SidePanelWidth
	if sideW <= 0 {
		sideW = 40
	}
	// Shrink side panel for medium terminals to fit [0] tabs.
	if a.width < 120 && sideW > a.width*35/100 {
		sideW = a.width * 35 / 100
	}
	if sideW > a.width/2 {
		sideW = a.width / 2
	}
	if sideW < 25 {
		sideW = 25
	}
	return sideW
}

func (a *App) layoutPanels() {
	totalH := a.height - 1

	if a.isVerticalLayout() {
		w := a.width
		logH := 5

		// Collapsed panels get 1 line; focused panel gets remaining space.
		statusH := 1  // status always collapsed in vertical
		issuesH := 1  // collapsed by default
		projectsH := 1 // collapsed by default
		var detailH int

		switch {
		case a.side == sideRight:
			// Detail focused: all left panels collapsed.
			detailH = totalH - statusH - issuesH - projectsH - logH
		case a.leftFocus == focusIssues:
			// Issues focused: expand issues, collapse others.
			detailH = 6
			issuesH = totalH - statusH - projectsH - logH - detailH
		case a.leftFocus == focusProjects:
			// Projects focused: expand projects, collapse others.
			detailH = 6
			projectsH = totalH - statusH - issuesH - logH - detailH
		case a.leftFocus == focusStatus:
			statusH = 3
			detailH = 6
			issuesH = totalH - statusH - projectsH - logH - detailH
		default:
			detailH = totalH - statusH - issuesH - projectsH - logH
		}

		if issuesH < 1 {
			issuesH = 1
		}
		if projectsH < 1 {
			projectsH = 1
		}
		if detailH < 3 {
			detailH = 3
		}

		a.statusPanel.SetSize(w, statusH)
		a.issuesList.SetSize(w, issuesH)
		a.projectList.SetSize(w, projectsH)
		a.detailView.SetSize(w, detailH)
		a.logPanel.SetSize(w, logH)
		a.panelSideW = w
		a.panelStatusH = statusH
		a.panelIssuesH = issuesH
		a.panelProjectsH = projectsH
		a.panelDetailH = detailH
		a.panelLogH = logH
		return
	}

	sideW := a.sideWidth()
	mainW := a.width - sideW

	statusH := 3
	remaining := totalH - statusH

	// Natural content heights (capped at remaining space).
	issuesNat := a.issuesList.ContentHeight()
	projectsNat := a.projectList.ContentHeight()
	minH := 3  // minimum panel height
	maxIssuesH := 12 // 10 items + 2 borders

	if issuesNat < minH {
		issuesNat = minH
	}
	if issuesNat > maxIssuesH {
		issuesNat = maxIssuesH
	}
	if projectsNat < minH {
		projectsNat = minH
	}

	var issuesH, projectsH int

	if issuesNat+projectsNat <= remaining {
		// Both fit — give extra space to the focused panel.
		extra := remaining - issuesNat - projectsNat
		if a.leftFocus == focusProjects {
			projectsH = projectsNat + extra
			issuesH = issuesNat
		} else {
			issuesH = issuesNat + extra
			projectsH = projectsNat
		}
	} else {
		// Not enough space — focused panel gets remaining, other gets natural or min.
		switch a.leftFocus {
		case focusIssues:
			projectsH = max(min(projectsNat, remaining/3), minH)
			issuesH = remaining - projectsH
		case focusProjects:
			issuesH = max(min(issuesNat, remaining/3), minH)
			projectsH = remaining - issuesH
		default:
			issuesH = remaining / 2
			projectsH = remaining - issuesH
		}
	}

	if issuesH < minH {
		issuesH = minH
	}
	if projectsH < minH {
		projectsH = minH
	}

	a.statusPanel.SetSize(sideW, statusH)
	a.issuesList.SetSize(sideW, issuesH)
	a.projectList.SetSize(sideW, projectsH)

	// Right column: log fits content or max 8, detail gets the rest.
	logH := 8
	detailH := max(totalH-logH, 5)

	a.detailView.SetSize(mainW, detailH)
	a.logPanel.SetSize(mainW, logH)

	// Cache sizes for mouse.
	a.panelSideW = sideW
	a.panelStatusH = statusH
	a.panelIssuesH = issuesH
	a.panelProjectsH = projectsH
	a.panelDetailH = detailH
	a.panelLogH = logH
}
