package secret

import (
	"context"
	"errors"
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

func (kubernetes *KubernetesSecretStorage) Persist(state vault.InitState) {
	ok, err := kubernetes.CreateSecret(objectOptions{
		dryRun: false,
		data:   state,
	})
	if !ok {
		panic(err.Error())
	}
}

func (kubernetes *KubernetesSecretStorage) Fetch() vault.InitState {
	return kubernetes.GetSecretData()
}

type objectOptions struct {
	dryRun bool
	data   vault.InitState
}

func (kubernetes *KubernetesSecretStorage) CreateSecret(options objectOptions) (bool, error) {
	dryRunOptions := []string{}
	if options.dryRun {
		dryRunOptions = append(dryRunOptions, metav1.DryRunAll)
	}

	ctx := context.Background()

	_, err := kubernetes.clientset.CoreV1().Secrets(kubernetes.namespace).Create(ctx,
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kubernetes.secretName,
				Namespace: kubernetes.namespace,
			},
			Data: encodeData(options.data),
		},
		metav1.CreateOptions{
			DryRun: dryRunOptions,
		})

	if err != nil {
		return false, err
	}
	return true, nil
}

func (kubernetes *KubernetesSecretStorage) GetSecretData() vault.InitState {
	ctx := context.Background()

	secret, err := kubernetes.clientset.CoreV1().Secrets(kubernetes.namespace).Get(ctx, kubernetes.secretName, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}

	return decodeData(secret.Data)
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
