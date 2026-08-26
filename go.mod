module github.com/dsh2dsh/expx-dotenv

go 1.24

require (
	github.com/joho/godotenv v1.5.1
	github.com/stretchr/testify v1.12.1
)

require go.yaml.in/yaml/v3 v3.0.5 // indirect

retract [v1.1.0, v1.3.0]
