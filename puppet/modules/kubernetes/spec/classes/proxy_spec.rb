require 'spec_helper'

describe 'kubernetes::proxy' do

  let :service_file do
      '/etc/systemd/system/kube-proxy.service'
  end

  let :proxy_config do
      '/etc/kubernetes/kube-proxy-config.yaml'
  end


  context 'proxy config' do
    context 'on kubernetes 1.10' do
      let(:pre_condition) {[
        """
        class{'kubernetes': version => '1.10.0'}
        """
      ]}
      it 'is not used' do
        should_not contain_file(service_file).with_content(%r{--config=/etc/kubernetes/kube-proxy-config\.yaml})
        should_not contain_file(proxy_config)
      end
    end

    context 'on kubernetes 1.11' do
      let(:pre_condition) {[
        """
        class{'kubernetes': version => '1.11.0'}
        """
      ]}
      it 'is used' do
          should contain_file(service_file).with_content(%r{--config=/etc/kubernetes/kube-proxy-config\.yaml})
          should contain_file(proxy_config)
      end

      context 'feature gates' do
        context 'none' do
          let(:pre_condition) {[
              """
              class{'kubernetes': enable_pod_priority => false}
              """
          ]}
          let(:params) { {
            "feature_gates" => []
          }}
          it 'none with no pod priority' do
            should_not contain_file(proxy_config).with_content(%r{featureGates:})
          end
        end

        context 'some' do
          let(:params) { {
            "feature_gates" => ["PodPriority=true", "foobar=true", "foo", "edge=case=true"]
          }}
          it 'config contain' do
            should contain_file(proxy_config).with_content(%r{featureGates:\n    PodPriority: true})
            should contain_file(proxy_config).with_content(%r{    foobar: true})
            should contain_file(proxy_config).with_content(%r{    foo: true})
            should contain_file(proxy_config).with_content(%r{    edge=case: true})
          end
        end
      end
    end
  end
end
