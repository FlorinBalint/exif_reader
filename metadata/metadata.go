package metadata

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

const exifTimeLayout = "2006:01:02 15:04:05"

var (
	timeType = reflect.TypeOf(time.Time{})
	ratType  = reflect.TypeOf(&big.Rat{})
)

type MD struct {
	Manufacturer     string    `exif_tag:"Make"`
	Model            string    `exif_tag:"Model"`
	LensManufacturer string    `exif_tag:"LensMake"`
	LensModel        string    `exif_tag:"LensModel"`
	DateTime         time.Time `exif_tag:"DateTimeOriginal"`
	FocalLengthMM    int64     `exif_tag:"FocalLengthIn35mmFilm"`
	ApertureFStop    *big.Rat  `exif_tag:"FNumber"`
	ShutterSpeed     *big.Rat  `exif_tag:"ExposureTime"`
	ISO              int64     `exif_tag:"ISOSpeedRatings"`
	SizeX            int64     `exif_tag:"PixelXDimension"`
	SizeY            int64     `exif_tag:"PixelYDimension"`
}

func (md *MD) String() string {
	var sBuilder strings.Builder
	sBuilder.WriteString(fmt.Sprintf("Taken on: %v\n", md.DateTime.Format("2006-01-02")))
	sBuilder.WriteString(fmt.Sprintf("Camera: %s %s\n", md.Manufacturer, md.Model))
	sBuilder.WriteString(fmt.Sprintf("Lens: %s %s\n", md.LensManufacturer, md.LensModel))
	sBuilder.WriteString(fmt.Sprintf("%dmm (35mm equivalent), f/%s %ss ISO %d\n",
		md.FocalLengthMM, md.ApertureFStop.FloatString(1), md.ShutterSpeed.RatString(), md.ISO))
	sBuilder.WriteString(fmt.Sprintf("Resolution: %dx%d ", md.SizeX, md.SizeY))
	return sBuilder.String()
}

func readField[T any](ex *exif.Exif, field exif.FieldName) (T, error) {
	var err error
	tag, err := ex.Get(field)
	resPt := new(T)

	if err != nil {
		return *resPt, err
	}

	switch {
	case tag.Format() == tiff.IntVal:
		aux, err := tag.Int64(0)
		if err != nil {
			return *resPt, err
		}
		*((*int64)(unsafe.Pointer(resPt))) = aux
	case tag.Format() == tiff.FloatVal:
		aux, err := tag.Float(0)
		if err != nil {
			return *resPt, err
		}
		*((*float64)(unsafe.Pointer(resPt))) = aux
	case tag.Format() == tiff.RatVal:
		aux, err := tag.Rat(0)
		if err != nil {
			return *resPt, err
		}
		*(**big.Rat)(unsafe.Pointer(resPt)) = aux
	case tag.Format() == tiff.StringVal:
		aux, err := tag.StringVal()
		if err != nil {
			return *resPt, err
		}
		*(*string)(unsafe.Pointer(resPt)) = aux
	default:
		return *resPt, fmt.Errorf("Unknwon field type %v for %q", tag.Format(), field)
	}

	return *resPt, err
}

func readMetadataField(ex *exif.Exif, mdField reflect.StructField) (reflect.Value, error) {
	tag := mdField.Tag
	exifField := exif.FieldName(tag.Get("exif_tag"))
	var reflectVal reflect.Value

	switch mdField.Type.Kind() {
	case reflect.Int64:
		val, err := readField[int64](ex, exifField)
		if err != nil {
			return reflectVal, err
		}
		reflectVal = reflect.ValueOf(val)
	case reflect.Float64:
		val, err := readField[float64](ex, exifField)
		if err != nil {
			return reflectVal, err
		}
		reflectVal = reflect.ValueOf(val)
	case reflect.String:
		val, err := readField[string](ex, exifField)
		if err != nil {
			return reflectVal, err
		}
		reflectVal = reflect.ValueOf(val)
	case reflect.Struct:
		if mdField.Type.AssignableTo(timeType) {
			valStr, err := readField[string](ex, exifField)
			if err != nil {
				return reflectVal, err
			}
			val, err := time.Parse(exifTimeLayout, valStr)
			if err != nil {
				return reflectVal, err
			}
			reflectVal = reflect.ValueOf(val)
		} else {
			return reflectVal, fmt.Errorf("Unknwon field type %T for %v", mdField, mdField)
		}
	case reflect.Pointer:
		if mdField.Type.AssignableTo(ratType) {
			val, err := readField[*big.Rat](ex, exifField)
			if err != nil {
				return reflectVal, err
			}
			reflectVal = reflect.ValueOf(val)
		} else {
			return reflectVal, fmt.Errorf("Unknwon field type %T for %v", mdField, mdField)
		}
	}
	return reflectVal, nil
}

func fromExif(ex *exif.Exif) (*MD, error) {
	md := MD{}
	mdType := reflect.TypeOf(md)
	mdVal := reflect.ValueOf(&md)

	for i := 0; i < mdType.NumField(); i++ {
		reflectVal, err := readMetadataField(ex, mdType.Field(i))
		if err != nil {
			if exif.IsTagNotPresentError(err) {
				log.Print(err)
				continue
			} else {
				return nil, err
			}
		}
		mdVal.Elem().Field(i).Set(reflectVal)
	}

	return &md, nil
}

func FromPhoto(file string) (*MD, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	x, err := exif.Decode(f)
	if err != nil {
		return nil, err
	}

	return fromExif(x)
}
