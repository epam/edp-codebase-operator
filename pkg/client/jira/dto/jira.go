package dto

type JiraServer struct {
	ApiUrl string
	User   string
	Pwd    string `json:"-"`
}

func ConvertSpecToJiraServer(url, user, password string) JiraServer {
	return JiraServer{
		ApiUrl: url,
		User:   user,
		Pwd:    password,
	}
}
