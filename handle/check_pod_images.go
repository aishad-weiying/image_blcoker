package handle

var (
	namespacesAdmit   = []string{"kube-system"}
	BlockerImagesList = []string{"nginx:latest"}
)

func MaditNS(ns string) bool {
	for _, aNs := range namespacesAdmit {
		if ns == aNs {
			return true
		}
	}
	return false
}

func MaditImageList(i string) bool {
	for _, blockerImage := range BlockerImagesList {
		if i == blockerImage {
			return false
		}
	}
	return true
}
