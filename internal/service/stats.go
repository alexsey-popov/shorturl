package service

// Stats отображает статистику сервиса
type Stats struct {
	Users int `json:"users"`
	Urls  int `json:"urls"`
}

// GetStats получает статистику сервиса
func (s Shortener) GetStats() (stats Stats, err error) {
	users, err := s.Rep.GetUsersCount()
	if err != nil {
		return
	}
	stats.Users = users

	urls, err := s.Rep.GetUrlsCount()
	if err != nil {
		return
	}
	stats.Urls = urls

	return
}
