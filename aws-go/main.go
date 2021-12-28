package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	MyBucket = ""
)

func Env(key string) string {
	return os.Getenv(key)
}

func init() {
	err := godotenv.Load("aws-go/.env")
	if err != nil {
		log.Panic(err)
	}

	MyBucket = Env("BUCKET_NAME")
}

func ConnectAWS() (*session.Session, error) {
	sess, err := session.NewSession(
		&aws.Config{
			Region: aws.String(Env("AWS_REGION")),
			Credentials: credentials.NewStaticCredentials(
				Env("AWS_ACCESS_KEY_ID"),
				Env("AWS_SECRET_ACCESS_KEY"),
				"",
			),
		})

	if err != nil {
		return nil, err
	}

	return sess, nil
}

func main() {
	fmt.Println(Env("AWS_SECRET_ACCESS_KEY"))
	sess, err := ConnectAWS()
	if err != nil {
		panic(err)
	}

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Set("sess", sess)
		c.Next()
	})

	router.POST("/upload", UploadImage)

	router.LoadHTMLGlob("aws-go/templates/*")
	router.GET("/download", Download)

	_ = router.Run(":4000")
}

func UploadImage(c *gin.Context) {
	sess := c.MustGet("sess").(*session.Session)
	file, header, err := c.Request.FormFile("photo")
	filename := header.Filename

	var buff bytes.Buffer
	fileSize, err := buff.ReadFrom(file)
	fileBuffer := make([]byte, fileSize)
	file.Read(fileBuffer)

	//upload to the s3 bucket
	uploader := s3manager.NewUploader(sess)
	up, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(MyBucket),
		Key:    aws.String(filename),
		ACL:    aws.String(s3.BucketCannedACLPublicRead),
		Body:   file,
	})

	//up, err := s3.New(sess).PutObjectAcl(&s3.PutObjectAclInput{
	//	ACL:              aws.String(s3.BucketCannedACLPrivate),
	//	Bucket:           aws.String(MyBucket),
	//	GrantFullControl: aws.String(s3.BucketCannedACLPrivate),
	//	Key:              aws.String(filename),
	//})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":    "Failed to upload file",
			"msg":      err.Error(),
			"uploader": up,
		})
		return
	}

	filepath := "https://" + MyBucket + "." + "s3-" + Env("AWS_REGION") + ".amazonaws.com/" + filename
	c.JSON(http.StatusOK, gin.H{
		"filepath": filepath,
	})
}

func Download(c *gin.Context) {
	item := "xxx.png"
	sess := c.MustGet("sess").(*session.Session)
	downloader := s3manager.NewDownloader(sess)

	file, err := os.Create(item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create file",
			"msg":   err.Error(),
		})
		return
	}

	numBytes, err := downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(MyBucket),
			Key:    aws.String(item),
		})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to download file",
			"msg":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": file.Name(),
		"bytes":    numBytes,
	})
	return
}
