# lipFP

This is LipFP, a set of fingerprint utilities. To be honest, I'm not entirely sure if it's for a fingerprint interprinter language or an intermec printer language, and I don't care anymore. So, I'm referring to it as "fingerprint" because that's what the screen of the printer says xD

If you can help me out here, please do so.

Use [pkg.go.dev](https://pkg.go.dev/github.com/rileys-trash-can/libfp) for documentation.

Useage / dokumentation `go install github.com/rileys-trash-can/libfp/cmd/fputils` to install.

Some of its capabilities include printing PNG/GIF/BMP images with a grayscale color model directly (printprbuf) or any other model by converting to RLL (printimg).

- `convert in.png -colormodel Gray out.png`

For the run-length-encoding based PRBUF image format, see `/prbuf`.
