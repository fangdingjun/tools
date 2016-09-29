/* convert the gif jpeg png image file to ico format

Author: dingjun<fangdingjun@gmail.com>
Date: 2016-9-29
License: GPLv3
*/
package main

import (
	"fmt"
	ico "github.com/Kodeworks/golang-image-ico"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Printf(
			"Usage: %s infile outfile\n\nconvert the image file infile to ico format and write to outfile\ncurrent support input image file format gif jpeg and png\n",
			os.Args[0])
		os.Exit(-1)
	}

	fp, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer fp.Close()

	// decode input image file
	im, _, err := image.Decode(fp)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	fpw, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer fpw.Close()

	// encode to icon format and write to file
	err = ico.Encode(fpw, im)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
