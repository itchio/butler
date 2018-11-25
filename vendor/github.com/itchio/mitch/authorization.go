package mitch

func (u *Upload) CanBeViewedBy(user *User) bool {
	// TODO: download keys etc.
	s := u.Store
	g := s.FindGame(u.GameID)
	if g == nil {
		Throw(404, "game not found")
	}

	return g.CanBeViewedBy(user)
}

func (g *Game) CanBeViewedBy(user *User) bool {
	if g.CanBeEditedBy(user) {
		return true
	}
	if g.Published {
		return true
	}
	return false
}

func (g *Game) CanBeEditedBy(user *User) bool {
	s := g.Store

	if g.UserID == user.ID {
		return true
	}
	admins := s.ListGameAdminsByGame(g.ID)
	for _, a := range admins {
		if a.UserID == user.ID {
			return true
		}
	}
	return false
}

func (u *Upload) CanBeDownloadedBy(user *User) bool {
	// TODO: download keys, min prices, etc.
	g := u.Store.FindGame(u.GameID)
	return g.CanBeViewedBy(user)
}
