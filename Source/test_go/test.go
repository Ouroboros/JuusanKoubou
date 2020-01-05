package main

type PutExtra struct {
    UpHost string
}

func put(e *PutExtra) error {
    println(e.UpHost)
    return nil
}

func main() {
    put(nil)
}
