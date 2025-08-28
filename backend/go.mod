module kube-node-manager

go 1.21

require (
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/go-ldap/ldap/v3 v3.4.6
	github.com/rakyll/statik v0.1.7
	gorm.io/driver/sqlite v1.5.4
	gorm.io/gorm v1.25.5
	k8s.io/api v0.28.3
	k8s.io/apimachinery v0.28.3
	k8s.io/client-go v0.28.3
	golang.org/x/crypto v0.15.0
	github.com/spf13/viper v1.17.0
)
