package secret

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"encoding/json"

	"github.com/mattgill98/vault-init/pkg/vault"
	v1 "k8s.io/api/core/v1"
	v1errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesSecretStorage struct {
	clientset  kubernetes.Interface
	namespace  string
	secretName string
}

var (
	ErrNotInCluster = errors.New("Kubernetes environment not detected")
)

func NewKubernetesSecretStorage(secretName string, namespace string) (KeyStorage, error) {
	// Construct Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, ErrNotInCluster
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	storage := &KubernetesSecretStorage{
		clientset:  clientset,
		namespace:  namespace,
		secretName: secretName,
	}

	// Test secret creation
	_, err = storage.CreateSecret(vault.InitState{Keys: []string{}, RootToken: ""})
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func (kubernetes *KubernetesSecretStorage) Persist(state vault.InitState) (bool, error) {
	ctx := context.Background()

	dataPatch, err := json.Marshal(v1.Secret{
		Data: encodeData(state),
	})
	if err != nil {
		return false, err
	}

	_, err = kubernetes.clientset.CoreV1().Secrets(kubernetes.namespace).Patch(ctx,
		kubernetes.secretName, types.StrategicMergePatchType, dataPatch, metav1.PatchOptions{})

	return err == nil, err
}

func (kubernetes *KubernetesSecretStorage) CreateSecret(state vault.InitState) (bool, error) {
	ctx := context.Background()

	_, err := kubernetes.clientset.CoreV1().Secrets(kubernetes.namespace).Create(ctx,
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubernetes.secretName,
				Namespace: kubernetes.namespace,
			},
			Data: encodeData(state),
		},
		metav1.CreateOptions{})

	if err == nil {
		return true, nil
	}

	// Ignore secret already exists - it will be updated
	if v1errors.IsAlreadyExists(err) {
		return false, nil
	}

	return false, err
}

func (kubernetes *KubernetesSecretStorage) Fetch() (*vault.InitState, error) {
	ctx := context.Background()

	secret, err := kubernetes.clientset.CoreV1().Secrets(kubernetes.namespace).Get(ctx, kubernetes.secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch secret data")
	}

	state := decodeData(secret.Data)
	return &state, nil
}

func encodeData(input vault.InitState) map[string][]byte {
	rootKeyBytes := []byte(input.RootToken)
	unsealKeysBytes := []byte(arrayToString(input.Keys))

	return map[string][]byte{
		"root_key":    rootKeyBytes,
		"unseal_keys": unsealKeysBytes,
	}
}

func decodeData(input map[string][]byte) vault.InitState {
	rootKey := string(input["root_key"])
	unsealKeys := string(input["unseal_keys"])

	return vault.InitState{
		RootToken: string(rootKey),
		Keys:      stringToArray(string(unsealKeys)),
	}
}

func arrayToString(input []string) string {
	return strings.Join(input, ",")
}

func stringToArray(input string) []string {
	return strings.Split(input, ",")
}
