package fixtures

var (
	UserOne                = []byte(`{"email": "test@mail.com", "password": "test"}`)
	UserOneUpdatedPassword = []byte(`{"email": "test@mail.com", "password":"new-password"}`)
	UserOneBadPassword     = []byte(`{"email": "test@mail.com", "password": "testerrrrr"}`)
	UserBadInput           = []byte(`{"gmail": "test@mail.com", "auth": "test", "extra_data": "data"}`)
	UserBadEmail           = []byte(`{"email": "test1mail.com", "password": "test"}`)

	LongUrl = []byte(`{"long_url":"https://www.google.com"}`)
)
