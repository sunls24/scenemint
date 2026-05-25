package main

import "scenemint/internal"

func main() {
	if err := internal.NewApp().Run(); err != nil {
		panic(err)
	}
}
