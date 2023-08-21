package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/h2non/bimg"
)

func main() {

	r := gin.Default()
	r.GET("/ping", ping)
	r.GET("/listings/:id/:md5AndTransforms", getImage)

	r.Run() // listen and serve on 0.0.0.0:8080
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Pong",
	})
}

type Hex string

type RGB struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

func Hex2RGB(hex Hex) (RGB, error) {
	var rgb RGB
	values, err := strconv.ParseUint(string(hex), 16, 32)

	if err != nil {
		return RGB{}, err
	}

	rgb = RGB{
		Red:   uint8(values >> 16),
		Green: uint8((values >> 8) & 0xFF),
		Blue:  uint8(values & 0xFF),
	}

	return rgb, nil
}

func getImage(c *gin.Context) {
	id, md5AndTransforms := c.Params.ByName("id"), c.Params.ByName("md5AndTransforms")

	fmt.Println(id, md5AndTransforms)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)

	md5 := md5AndTransforms[:32]
	transforms := md5AndTransforms[32:]

	transformParts := strings.Split(transforms, ".")

	transformation := transformParts[1]
	var background RGB
	if strings.Contains(transformation, "pad") {
		hex := Hex(transformation[4:])
		transformation = "pad"
		background, err = Hex2RGB(hex)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(background)
	}

	dimensions := transformParts[2]
	fileType := transformParts[3]

	dimensionsParts := strings.Split(dimensions, "x")

	width, err := strconv.Atoi(dimensionsParts[0])
	height, err := strconv.Atoi(dimensionsParts[1])

	// Get the first page of results for ListObjectsV2 for a bucket
	output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("SOURCE_BUCKETS")),
		Key:    aws.String("listings/" + md5 + "." + fileType),
	})
	if err != nil {
		log.Fatal(err)
	}

	buffer, err := io.ReadAll(output.Body)

	outputType := bimg.ImageType(bimg.JPEG)
	if os.Getenv("ENABLE_WEBP") == "true" {
		outputType = bimg.ImageType(bimg.WEBP)
	}

	options := bimg.Options{
		Width:  width,
		Height: height,
		Type:   outputType,
	}
	if transformation == "crop" {
		options.Crop = true
		options.Gravity = bimg.GravityCentre
	}
	if transformation == "fit" {
		options.Force = true
	}
	if transformation == "scale" {
		options.Enlarge = true
	}
	if transformation == "pad" {
		options.Embed = true
		options.Extend = bimg.ExtendBackground
		options.Background = bimg.Color{R: background.Red, G: background.Green, B: background.Blue}
	}

	newImage, err := bimg.NewImage(buffer).Process(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	c.Header("Content-Type", *output.ContentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(newImage)))
	c.Status(http.StatusOK)

	c.Data(http.StatusOK, *output.ContentType, newImage)
}
