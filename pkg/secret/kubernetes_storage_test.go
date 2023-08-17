package secret

import (
	"testing"

	b64 "encoding/base64"

	"github.com/mattgill98/vault-init/pkg/vault"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetSecretData(t *testing.T) {
	encoder := b64.StdEncoding.Strict()
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-secret",
			Namespace: "demo",
		},
		Data: map[string][]byte{
			"root_key":    []byte(encoder.EncodeToString([]byte("abc"))),
			"unseal_keys": []byte(encoder.EncodeToString([]byte("a,b,c"))),
		},
	}

	clientset := fake.NewSimpleClientset(&secret)

	storage := KubernetesSecretStorage{
		clientset:  clientset,
		namespace:  "demo",
		secretName: "demo-secret",
	}

	data := storage.GetSecretData()

	assert.Equal(t, "abc", data.RootToken)
	assert.Equal(t, []string{"a", "b", "c"}, data.Keys)
}

func TestCreateSecret(t *testing.T) {
	encoder := b64.StdEncoding.Strict()
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-secret",
			Namespace: "demo",
		},
		Data: map[string][]byte{
			"root_key":    []byte(encoder.EncodeToString([]byte("abc"))),
			"unseal_keys": []byte(encoder.EncodeToString([]byte("a,b,c"))),
		},
	}

	clientset := fake.NewSimpleClientset()

	storage := KubernetesSecretStorage{
		clientset:  clientset,
		namespace:  "demo",
		secretName: "demo-secret",
	}

	ok, err := storage.CreateSecret(objectOptions{data: vault.InitState{Keys: []string{"a", "b", "c"}, RootToken: "abc"}})
	assert.True(t, ok)
	assert.Nil(t, err)

	object, err := clientset.Tracker().Get(v1.SchemeGroupVersion.WithResource("secrets"), "demo", "demo-secret")
	assert.Equal(t, secret.Data, (object.(*v1.Secret).Data))
}
