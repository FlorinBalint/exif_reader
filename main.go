package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

type Printer struct{}

func (p Printer) Walk(name exif.FieldName, tag *tiff.Tag) error {
	fmt.Printf("%40s: %s\n", name, tag)
	return nil
}

type Metadata struct {
	Manufacturer     string
	Model            string
	LensManufacturer string
	LensModel        string
	DateTime         time.Time
	FocalLengthMM    int64
	ApertureFStop    *big.Rat
	ShutterSpeed     *big.Rat
	ISO              int64
	SizeX            int64
	SizeY            int64
}

func (md *Metadata) String() string {
	var sBuilder strings.Builder
	sBuilder.WriteString(fmt.Sprintf("Taken on: %v\n", md.DateTime.Format("2006-01-02")))
	sBuilder.WriteString(fmt.Sprintf("Camera: %s %s\n", md.Manufacturer, md.Model))
	sBuilder.WriteString(fmt.Sprintf("Lens: %s %s\n", md.LensManufacturer, md.LensModel))
	sBuilder.WriteString(fmt.Sprintf("%dmm (35mm equivalent), f/%s %ss ISO %d\n",
		md.FocalLengthMM, md.ApertureFStop.FloatString(1), md.ShutterSpeed.RatString(), md.ISO))
	sBuilder.WriteString(fmt.Sprintf("Resolution: %dx%d ", md.SizeX, md.SizeY))
	return sBuilder.String()
}

func readFraction(ex *exif.Exif, field exif.FieldName) (*big.Rat, error) {
	tif, err := ex.Get(field)
	if err != nil {
		return nil, err
	}
	return tif.Rat(0)
}

func readString(ex *exif.Exif, field exif.FieldName) (string, error) {
	tif, err := ex.Get(field)
	if err != nil {
		return "", err
	}
	return tif.StringVal()
}

func readInt64(ex *exif.Exif, field exif.FieldName) (int64, error) {
	tif, err := ex.Get(field)
	if err != nil {
		return -1, err
	}
	return tif.Int64(0)
}

func fromExif(ex *exif.Exif) (*Metadata, error) {
	md := &Metadata{}
	md.DateTime, _ = ex.DateTime()
	md.Manufacturer, _ = readString(ex, exif.Make)
	md.Model, _ = readString(ex, exif.Model)
	md.LensManufacturer, _ = readString(ex, exif.LensMake)
	md.LensModel, _ = readString(ex, exif.LensModel)
	md.SizeX, _ = readInt64(ex, exif.PixelXDimension)
	md.SizeY, _ = readInt64(ex, exif.PixelYDimension)
	md.FocalLengthMM, _ = readInt64(ex, exif.FocalLengthIn35mmFilm)
	md.ISO, _ = readInt64(ex, exif.ISOSpeedRatings)
	md.ApertureFStop, _ = readFraction(ex, exif.FNumber)
	md.ShutterSpeed, _ = readFraction(ex, exif.ExposureTime)
	return md, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("please give filename as argument")
	}
	fname := os.Args[1]

	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}

	x, err := exif.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	var p Printer
	x.Walk(p)

	md, err := fromExif(x)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Metadata:\n%v", md)
}
