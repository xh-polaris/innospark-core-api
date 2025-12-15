package conf

type Mongo struct {
	URL string
	DB  string
}

type Cache struct {
	Addr     string
	Password string
}

type COS struct {
	AppID     string
	BucketURL string
	CDN       string
	SecretID  string
	SecretKey string
}

type Auth struct {
	SecretKey    string
	PublicKey    string
	AccessExpire int64
}
