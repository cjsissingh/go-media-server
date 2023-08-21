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

func getImage(c *gin.Context) {
	md5AndTransforms := c.Params.ByName("md5AndTransforms")
	if md5AndTransforms == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"status": http.StatusBadRequest,
			"code": "DecodeRequest::CannotDecodeRequest",
			"message": "The URL does not have enough parts. The format must be '/{resource}/{resourceId}/{md5}.{auto|scale|crop|pad-{colour}}.{width}x{height}.jpg'",
			"level": "error",
		})
		return
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	// Create an Amazon S3 service client
	client := s3.NewFromConfig(cfg)

	md5 := md5AndTransforms[:32]
	transforms := md5AndTransforms[32:]

	transformParts := strings.Split(transforms, ".")
	if len(transformParts) != 4 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"status": http.StatusBadRequest,
			"code": "DecodeRequest::CannotDecodeRequest",
			"message": "The URL does not have enough parts. The format must be '/{resource}/{resourceId}/{md5}.{auto|scale|crop|pad-{colour}}.{width}x{height}.jpg'",
			"level": "error",
		})
		return
	}

	transformation := transformParts[1]

	dimensions := transformParts[2]
	fileType := transformParts[3]

	dimensionsParts := strings.Split(dimensions, "x")

	width, err := strconv.Atoi(dimensionsParts[0])
	height, err := strconv.Atoi(dimensionsParts[1])

	output, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("SOURCE_BUCKETS")),
		Key:    aws.String("listings/" + md5 + "." + fileType),
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"status": http.StatusNotFound,
			"message": "The specified key does not exist.",
			"level": "error",
		})
		return
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

	// Not a supported option
	// if transformation == "fit" {
	// 	options.Force = true
	// }

	if transformation == "scale" {
		options.Enlarge = true
	}

	if strings.Contains(transformation, "pad")  {
		hexString := transformation[4:]
		if len(hexString) == 3 {
			hexString = hexString[0:1] + hexString[0:1] + hexString[1:2] + hexString[1:2] + hexString[2:3] + hexString[2:3]
		}

		hex := Hex(hexString)
		transformation = "pad"

		background, err := Hex2RGB(hex)
		if err != nil {
			log.Fatal(err)
		}
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
