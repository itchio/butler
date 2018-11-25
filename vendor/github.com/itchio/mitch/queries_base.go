package mitch

func (s *Store) SelectGame(vsb *ValuesSortBuilder, eq Eq) *Game {
	var res Game
	if s.SelectOne(&res, vsb.ForMap(s.Games), eq) {
		return &res
	} else {
		return nil
	}
}

func (s *Store) SelectGames(vsb *ValuesSortBuilder, eq Eq) (res []*Game) {
	s.Select(&res, vsb.ForMap(s.Games), eq)
	return
}

func (s *Store) SelectUpload(vsb *ValuesSortBuilder, eq Eq) *Upload {
	var res Upload
	if s.SelectOne(&res, vsb.ForMap(s.Uploads), eq) {
		return &res
	} else {
		return nil
	}
}

func (s *Store) SelectUploads(vsb *ValuesSortBuilder, eq Eq) (res []*Upload) {
	s.Select(&res, vsb.ForMap(s.Uploads), eq)
	return
}

func (s *Store) SelectBuild(vsb *ValuesSortBuilder, eq Eq) *Build {
	var res Build
	if s.SelectOne(&res, vsb.ForMap(s.Builds), eq) {
		return &res
	} else {
		return nil
	}
}

func (s *Store) SelectBuilds(vsb *ValuesSortBuilder, eq Eq) (res []*Build) {
	s.Select(&res, vsb.ForMap(s.Builds), eq)
	return
}

func (s *Store) SelectAPIKey(vsb *ValuesSortBuilder, eq Eq) *APIKey {
	var res APIKey
	if s.SelectOne(&res, vsb.ForMap(s.APIKeys), eq) {
		return &res
	} else {
		return nil
	}
}

func (s *Store) SelectAPIKeys(vsb *ValuesSortBuilder, eq Eq) (res []*APIKey) {
	s.Select(&res, vsb.ForMap(s.APIKeys), eq)
	return
}

func (s *Store) SelectUser(vsb *ValuesSortBuilder, eq Eq) *User {
	var res User
	if s.SelectOne(&res, vsb.ForMap(s.Users), eq) {
		return &res
	} else {
		return nil
	}
}

func (s *Store) SelectUsers(vsb *ValuesSortBuilder, eq Eq) (res []*User) {
	s.Select(&res, vsb.ForMap(s.Users), eq)
	return
}

func (s *Store) SelectBuildFile(vsb *ValuesSortBuilder, eq Eq) *BuildFile {
	var res BuildFile
	if s.SelectOne(&res, vsb.ForMap(s.BuildFiles), eq) {
		return &res
	} else {
		return nil
	}
}

func (s *Store) SelectBuildFiles(vsb *ValuesSortBuilder, eq Eq) (res []*BuildFile) {
	s.Select(&res, vsb.ForMap(s.BuildFiles), eq)
	return
}

func (s *Store) SelectGameAdmin(vsb *ValuesSortBuilder, eq Eq) *GameAdmin {
	var res GameAdmin
	if s.SelectOne(&res, vsb.ForMap(s.GameAdmins), eq) {
		return &res
	} else {
		return nil
	}
}

func (s *Store) SelectGameAdmins(vsb *ValuesSortBuilder, eq Eq) (res []*GameAdmin) {
	s.Select(&res, vsb.ForMap(s.GameAdmins), eq)
	return
}
