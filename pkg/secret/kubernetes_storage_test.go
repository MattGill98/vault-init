package secret

import (
	"testing"

	"github.com/mattgill98/vault-init/pkg/vault"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetSecretData(t *testing.T) {
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-secret",
			Namespace: "demo",
		},
		Data: map[string][]byte{
			"root_key":    []byte([]byte("abc")),
			"unseal_keys": []byte([]byte("a,b,c")),
		},
	}

	clientset := fake.NewSimpleClientset(&secret)

	storage := KubernetesSecretStorage{
		clientset:  clientset,
		namespace:  "demo",
		secretName: "demo-secret",
	}

	state, err := storage.Fetch()

	assert.Nil(t, err)
	assert.Equal(t, "abc", state.RootToken)
	assert.Equal(t, []string{"a", "b", "c"}, state.Keys)
}

func TestUpdateSecret(t *testing.T) {
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo-secret",
			Namespace: "demo",
		},
		Data: map[string][]byte{
			"root_key":    []byte([]byte("abc")),
			"unseal_keys": []byte([]byte("a,b,c")),
		},
	}

	clientset := fake.NewSimpleClientset(&secret)

	storage := KubernetesSecretStorage{
		clientset:  clientset,
		namespace:  "demo",
		secretName: "demo-secret",
	}

	ok, err := storage.Persist(vault.InitState{Keys: []string{"a", "b", "c"}, RootToken: "abc"})
	assert.True(t, ok)
	assert.Nil(t, err)

	object, err := clientset.Tracker().Get(v1.SchemeGroupVersion.WithResource("secrets"), "demo", "demo-secret")
	assert.Equal(t, secret.Data, (object.(*v1.Secret).Data))
}
