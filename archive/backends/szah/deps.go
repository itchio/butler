package szah

import "runtime"

type DepSpec struct {
	entries []DepEntry
	sources []string
}

type DepEntry struct {
	name   string
	hashes []DepHash
}

type HashAlgo string

const (
	HashAlgoSHA1   = "sha1"
	HashAlgoSHA256 = "sha256"
)

type DepHash struct {
	algo HashAlgo
	// byte array formatted with `%x` (lower-case hex)
	value string
}

func getDepSpec() *DepSpec {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "386":
			return &DepSpec{
				entries: []DepEntry{
					DepEntry{
						name: "7z.dll",
						hashes: []DepHash{
							DepHash{
								algo:  HashAlgoSHA1,
								value: "",
							},
							DepHash{
								algo:  HashAlgoSHA256,
								value: "",
							},
						},
					},
					DepEntry{
						name: "c7zip.dll",
						hashes: []DepHash{
							DepHash{
								algo:  HashAlgoSHA1,
								value: "",
							},
							DepHash{
								algo:  HashAlgoSHA256,
								value: "",
							},
						},
					},
				},
				sources: []string{
					"https://dl.itch.ovh/libc7zip/windows-386/v1.1.0/libc7zip.zip",
				},
			}
		case "amd64":
			return &DepSpec{
				entries: []DepEntry{
					DepEntry{
						name: "7z.dll",
						hashes: []DepHash{
							DepHash{
								algo:  HashAlgoSHA1,
								value: "c2c0b3e5cadcff3b747c856939b1785c34a13435",
							},
							DepHash{
								algo:  HashAlgoSHA256,
								value: "fdfdc8419d1911e389d150e74f1f7a5277eed4e300f04176756d75ad862a6188",
							},
						},
					},
					DepEntry{
						name: "c7zip.dll",
						hashes: []DepHash{
							DepHash{
								algo:  HashAlgoSHA1,
								value: "4c6bf298d939478e076aed0084b78a073aea4cd7",
							},
							DepHash{
								algo:  HashAlgoSHA256,
								value: "c70c042c6ad017adeb74acb72ded6e698cccaedccf9dd39e0ed926ce043f8b28",
							},
						},
					},
				},
				sources: []string{
					"https://dl.itch.ovh/libc7zip/windows-amd64/v1.1.0/libc7zip.zip",
				},
			}
		}
	case "darwin":
		return nil
	case "linux":
		return nil
	}

	return nil
}
