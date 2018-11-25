package mitch

func (s *Store) FindAPIKeysByKey(key string) *APIKey {
	return s.SelectAPIKey(NoSort(), Eq{"Key": key})
}

func (s *Store) ListAPIKeysByUser(userID int64) []*APIKey {
	return s.SelectAPIKeys(NoSort(), Eq{"UserID": userID})
}

func (s *Store) FindUser(id int64) *User {
	return s.Users[id]
}

func (s *Store) FindGame(id int64) *Game {
	return s.Games[id]
}

func (s *Store) FindUpload(id int64) *Upload {
	return s.Uploads[id]
}

func (s *Store) FindBuild(id int64) *Build {
	return s.Builds[id]
}

func (s *Store) ListUploadsByGame(gameID int64) []*Upload {
	return s.SelectUploads(NoSort(), Eq{"GameID": gameID})
}

func (s *Store) ListBuildsByUpload(uploadID int64) []*Build {
	return s.SelectBuilds(SortBy("ID", "desc"), Eq{"UploadID": uploadID})
}

func (s *Store) ListGameAdminsByGame(gameID int64) []*GameAdmin {
	return s.SelectGameAdmins(NoSort(), Eq{"GameID": gameID})
}
