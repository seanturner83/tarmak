package stack

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	vaultUnsealer "github.com/jetstack-experimental/vault-unsealer/pkg/vault"

	"github.com/jetstack/tarmak/pkg/tarmak/config"
	"github.com/jetstack/tarmak/pkg/tarmak/interfaces"
)

var vaultClientLock sync.Mutex

type VaultStack struct {
	*Stack
}

var _ interfaces.Stack = &VaultStack{}

func newVaultStack(s *Stack, conf *config.StackVault) (*VaultStack, error) {
	v := &VaultStack{
		Stack: s,
	}

	s.name = config.StackNameVault
	s.verifyPost = append(s.verifyPost, v.verifyVaultInit)
	return v, nil
}

func (s *VaultStack) Variables() map[string]interface{} {
	return map[string]interface{}{}
}

const (
	VaultStateSealed = iota
	VaultStateUnsealed
	VaultStateUnintialised
	VaultStateErr
)

type vaultTunnel struct {
	tunnel      interfaces.Tunnel
	tunnelError error
	client      *vault.Client
	fqdn        string
}

func (s *VaultStack) vaultCA() ([]byte, error) {
	vaultCAIntf, ok := s.output["vault_ca"]
	if !ok {
		return []byte{}, fmt.Errorf("unable to find terraform output 'vault_ca'")
	}

	vaultCA, ok := vaultCAIntf.(string)
	if !ok {
		return []byte{}, fmt.Errorf("unexpected type for 'vault_ca': %t", vaultCAIntf)
	}

	return []byte(vaultCA), nil
}

func (s *VaultStack) vaultURL() (*url.URL, error) {
	key := "vault_url"
	vaultURLIntf, ok := s.output[key]
	if !ok {
		return nil, fmt.Errorf("unable to find terraform output '%s'", key)
	}

	vaultURL, ok := vaultURLIntf.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for '%s': %T", key, vaultURLIntf)
	}

	url, err := url.Parse(vaultURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing vault url '%s': %s", vaultURL, err)
	}
	return url, nil

}

func (s *VaultStack) vaultInstanceFQDNs() ([]string, error) {
	instanceFQDNsIntf, ok := s.output["instance_fqdns"]
	if !ok {
		return []string{}, fmt.Errorf("unable to find terraform output 'instance_fqdns'")
	}

	instanceFQDNsInftSlice, ok := instanceFQDNsIntf.([]interface{})
	if !ok {
		return []string{}, fmt.Errorf("unexpected type for 'instance_fqdns': %T", instanceFQDNsIntf)
	}

	instanceFQDNs := make([]string, len(instanceFQDNsInftSlice))
	for pos, value := range instanceFQDNsInftSlice {
		var ok bool
		instanceFQDNs[pos], ok = value.(string)
		if !ok {
			return []string{}, fmt.Errorf("unexpected type for element %d in 'instance_fqdns': %T", pos, value)
		}
	}

	return instanceFQDNs, nil
}

// This returns a vault tunnel for the whole cluster (using DNS RR)
func (s *VaultStack) VaultTunnel() (*vaultTunnel, error) {
	instance, err := s.vaultURL()
	if err != nil {
		return nil, err
	}

	tunnels, err := s.createVaultTunnels([]string{strings.Split(instance.Host, ":")[0]})
	return tunnels[0], err
}

// This returns a vault tunnel per instance
func (s *VaultStack) VaultTunnels() ([]*vaultTunnel, error) {
	vaultInstances, err := s.vaultInstanceFQDNs()
	if err != nil {
		return []*vaultTunnel{}, fmt.Errorf("couldn't load vault instance fqdns from terraform: %s", err)
	}

	return s.createVaultTunnels(vaultInstances)
}

func (s *VaultStack) createVaultTunnels(instances []string) ([]*vaultTunnel, error) {
	vaultCA, err := s.vaultCA()
	if err != nil {
		return []*vaultTunnel{}, fmt.Errorf("couldn't load vault CA from terraform: %s", err)
	}

	tlsConfig := &tls.Config{RootCAs: x509.NewCertPool()}

	ok := tlsConfig.RootCAs.AppendCertsFromPEM(vaultCA)
	if !ok {
		return []*vaultTunnel{}, fmt.Errorf("couldn't load vault CA certificate into http client")
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	httpClient := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	vaultClient, err := vault.NewClient(&vault.Config{
		HttpClient: httpClient,
	})
	if err != nil {
		return []*vaultTunnel{}, fmt.Errorf("couldn't init vault client: %s:", err)
	}

	output := make([]*vaultTunnel, len(instances))
	for pos, _ := range instances {
		output[pos] = s.newVaultTunnel(instances[pos], vaultClient)
	}

	return output, nil

}

func (s *VaultStack) newVaultTunnel(fqdn string, client *vault.Client) *vaultTunnel {
	return &vaultTunnel{
		tunnel: s.Context().Environment().Tarmak().SSH().Tunnel("bastion", fqdn, 8200),
		client: client,
		fqdn:   fqdn,
	}
}

func (v *vaultTunnel) FQDN() string {
	return v.fqdn
}
func (v *vaultTunnel) Start() error {

	if err := v.tunnel.Start(); err != nil {
		v.tunnelError = err
		return err
	}

	return nil
}

func (v *vaultTunnel) Stop() error {
	return v.tunnel.Stop()
}

func (v *vaultTunnel) VaultClient() *vault.Client {
	v.client.SetAddress(fmt.Sprintf("https://localhost:%d", v.tunnel.Port()))
	return v.client
}

func (v *vaultTunnel) Status() int {
	if v.tunnelError != nil {
		return VaultStateErr
	}

	vaultClientLock.Lock()
	defer vaultClientLock.Unlock()

	v.VaultClient()

	initStatus, err := v.client.Sys().InitStatus()
	if err != nil {
		return VaultStateErr
	}

	if !initStatus {
		return VaultStateUnintialised
	}

	sealStatus, err := v.client.Sys().SealStatus()
	if err != nil {
		return VaultStateErr
	}

	if sealStatus.Sealed {
		return VaultStateSealed
	}
	return VaultStateUnsealed
}

func (s *VaultStack) vaultInstanceState(tunnels []*vaultTunnel) (state int, instances []*vaultTunnel) {
	instanceState := map[int][]*vaultTunnel{}
	for pos, _ := range tunnels {
		state := tunnels[pos].Status()
		if _, ok := instanceState[state]; !ok {
			instanceState[state] = []*vaultTunnel{tunnels[pos]}
		} else {
			instanceState[state] = append(instanceState[state], tunnels[pos])
		}
		s.log.Debugf("vault %s status: %d", tunnels[pos].FQDN(), tunnels[pos].Status())
	}

	// get state that has quorum
	for state, instances := range instanceState {
		if len(instances) > len(tunnels)/2 {
			return state, instances
		}
	}
	return VaultStateErr, []*vaultTunnel{}
}

func (s *VaultStack) verifyVaultInit() error {

	tunnels, err := s.VaultTunnels()
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	for pos, _ := range tunnels {
		wg.Add(1)
		go func(pos int) {
			defer wg.Done()
			err := tunnels[pos].Start()
			if err != nil {
				s.log.Warn(err)
			}
		}(pos)
	}

	// wait for all tunnel attempts
	wg.Wait()

	defer func() {
		var wg sync.WaitGroup
		for pos, _ := range tunnels {
			wg.Add(1)
			go func(pos int) {
				defer wg.Done()
				err := tunnels[pos].Stop()
				if err != nil {
					s.log.Warn(err)
				}
			}(pos)
		}
		wg.Wait()
	}()

	// get state of all instances
	retries := 60
	for {
		clusterState, instances := s.vaultInstanceState(tunnels)

		if clusterState == VaultStateUnsealed {
			// quorum of vaults is unsealed
			return nil
		} else if clusterState == VaultStateUnintialised {
			rootToken, err := s.Context().Environment().VaultRootToken()
			if err != nil {
				return err
			}

			kv, err := s.Context().Environment().Provider().VaultKV()
			if err != nil {
				return err
			}

			vaultClientLock.Lock()
			defer vaultClientLock.Unlock()

			cl := instances[0].VaultClient()

			v, err := vaultUnsealer.New(kv, cl, vaultUnsealer.Config{
				KeyPrefix: "vault",

				SecretShares:    1,
				SecretThreshold: 1,

				InitRootToken:  rootToken,
				StoreRootToken: false,

				OverwriteExisting: true,
			})

			err = v.Init()
			if err != nil {
				return fmt.Errorf("error initialising vault: %s", err)
			}
			s.log.Info("vault succesfully initialised")
			return nil

		} else if clusterState == VaultStateSealed {
			s.log.Debug("a quorum of vault instances is sealed, retrying")
		} else {
			s.log.Debug("a quorum of vault instances is in unknown state, retrying")
		}
		retries -= 1
		if retries == 0 {
			return fmt.Errorf("time out verifying that vault cluster is initialiased and unsealed")
		}
		time.Sleep(time.Second * 10)
	}

	return nil
}