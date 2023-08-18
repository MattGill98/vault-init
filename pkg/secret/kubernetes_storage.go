package secret

import (
	"context"
	"errors"
	"fmt"
	"strings"

	b64 "encoding/base64"

	"github.com/mattgill98/vault-init/pkg/vault"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	return &KubernetesSecretStorage{
		clientset: clientset,
		namespace: namespace,
	}, nil
}

func (kubernetes *KubernetesSecretStorage) Persist(state vault.InitState) (bool, error) {
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

	return err == nil, err
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
	encoder := b64.StdEncoding.Strict()

	rootKeyBytes := []byte(input.RootToken)
	encodedRootKeyBytes := []byte(encoder.EncodeToString(rootKeyBytes))

	unsealKeysBytes := []byte(arrayToString(input.Keys))
	encodedUnsealKeysBytes := []byte(encoder.EncodeToString(unsealKeysBytes))

	return map[string][]byte{
		"root_key":    encodedRootKeyBytes,
		"unseal_keys": encodedUnsealKeysBytes,
	}
}

func decodeData(input map[string][]byte) vault.InitState {
	encoder := b64.StdEncoding.Strict()

	rootKey, err := encoder.DecodeString(string(input["root_key"]))
	if err != nil {
		panic(err)
	}
	unsealKeys, err := encoder.DecodeString(string(input["unseal_keys"]))
	if err != nil {
		panic(err)
	}

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
