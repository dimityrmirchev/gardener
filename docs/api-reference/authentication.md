<p>Packages:</p>
<ul>
<li>
<a href="#authentication.gardener.cloud%2fv1alpha1">authentication.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="authentication.gardener.cloud/v1alpha1">authentication.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 is a version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#authentication.gardener.cloud/v1alpha1.WorkloadIdentity">WorkloadIdentity</a>
</li></ul>
<h3 id="authentication.gardener.cloud/v1alpha1.WorkloadIdentity">WorkloadIdentity
</h3>
<p>
<p>WorkloadIdentity holds certain properties related to Gardener managed workload communicating with external systems.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
authentication.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>WorkloadIdentity</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Standard object metadata.</p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#authentication.gardener.cloud/v1alpha1.WorkloadIdentitySpec">
WorkloadIdentitySpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Spec defines the workload identity properties.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>audiences</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Audiences represent the target systems which the current workload identity will be used against.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#authentication.gardener.cloud/v1alpha1.WorkloadIdentityStatus">
WorkloadIdentityStatus
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Most recently observed status of the WorkloadIdentity.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.AdminKubeconfigRequest">AdminKubeconfigRequest
</h3>
<p>
<p>AdminKubeconfigRequest can be used to request a kubeconfig with admin credentials
for a Shoot cluster.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard object metadata.</p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#authentication.gardener.cloud/v1alpha1.AdminKubeconfigRequestSpec">
AdminKubeconfigRequestSpec
</a>
</em>
</td>
<td>
<p>Spec is the specification of the AdminKubeconfigRequest.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>expirationSeconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExpirationSeconds is the requested validity duration of the credential. The
credential issuer may return a credential with a different validity duration so a
client needs to check the &lsquo;expirationTimestamp&rsquo; field in a response.
Defaults to 1 hour.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#authentication.gardener.cloud/v1alpha1.AdminKubeconfigRequestStatus">
AdminKubeconfigRequestStatus
</a>
</em>
</td>
<td>
<p>Status is the status of the AdminKubeconfigRequest.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.AdminKubeconfigRequestSpec">AdminKubeconfigRequestSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#authentication.gardener.cloud/v1alpha1.AdminKubeconfigRequest">AdminKubeconfigRequest</a>)
</p>
<p>
<p>AdminKubeconfigRequestSpec contains the expiration time of the kubeconfig.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>expirationSeconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExpirationSeconds is the requested validity duration of the credential. The
credential issuer may return a credential with a different validity duration so a
client needs to check the &lsquo;expirationTimestamp&rsquo; field in a response.
Defaults to 1 hour.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.AdminKubeconfigRequestStatus">AdminKubeconfigRequestStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#authentication.gardener.cloud/v1alpha1.AdminKubeconfigRequest">AdminKubeconfigRequest</a>)
</p>
<p>
<p>AdminKubeconfigRequestStatus is the status of the AdminKubeconfigRequest containing
the kubeconfig and expiration of the credential.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>kubeconfig</code></br>
<em>
[]byte
</em>
</td>
<td>
<p>Kubeconfig contains the kubeconfig with cluster-admin privileges for the shoot cluster.</p>
</td>
</tr>
<tr>
<td>
<code>expirationTimestamp</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>ExpirationTimestamp is the expiration timestamp of the returned credential.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.TokenRequest">TokenRequest
</h3>
<p>
<p>TokenRequest can be used to request a token with for a specific workload identity.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<p>Standard object metadata.</p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#authentication.gardener.cloud/v1alpha1.TokenRequestRequestSpec">
TokenRequestRequestSpec
</a>
</em>
</td>
<td>
<p>Spec is the specification of the TokenRequest.</p>
<br/>
<br/>
<table>
<tr>
<td>
<code>expirationSeconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExpirationSeconds is the requested validity duration of the credential. The
credential issuer may return a credential with a different validity duration so a
client needs to check the &lsquo;expirationTimestamp&rsquo; field in a response.
Defaults to 1 hour.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#authentication.gardener.cloud/v1alpha1.TokenRequestStatus">
TokenRequestStatus
</a>
</em>
</td>
<td>
<p>Status is the status of the TokenRequest.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.TokenRequestRequestSpec">TokenRequestRequestSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#authentication.gardener.cloud/v1alpha1.TokenRequest">TokenRequest</a>)
</p>
<p>
<p>TokenRequestRequestSpec contains the expiration time of the token.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>expirationSeconds</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExpirationSeconds is the requested validity duration of the credential. The
credential issuer may return a credential with a different validity duration so a
client needs to check the &lsquo;expirationTimestamp&rsquo; field in a response.
Defaults to 1 hour.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.TokenRequestStatus">TokenRequestStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#authentication.gardener.cloud/v1alpha1.TokenRequest">TokenRequest</a>)
</p>
<p>
<p>TokenRequestStatus is the status of the TokenRequest containing
the token.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>token</code></br>
<em>
string
</em>
</td>
<td>
<p>Token is the bearer token.</p>
</td>
</tr>
<tr>
<td>
<code>expirationTimestamp</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<p>ExpirationTimestamp is the expiration timestamp of the returned credential.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.WorkloadIdentitySpec">WorkloadIdentitySpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#authentication.gardener.cloud/v1alpha1.WorkloadIdentity">WorkloadIdentity</a>)
</p>
<p>
<p>WorkloadIdentitySpec is the specification of a WorkloadIdentity.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>audiences</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Audiences represent the target systems which the current workload identity will be used against.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="authentication.gardener.cloud/v1alpha1.WorkloadIdentityStatus">WorkloadIdentityStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#authentication.gardener.cloud/v1alpha1.WorkloadIdentity">WorkloadIdentity</a>)
</p>
<p>
<p>WorkloadIdentityStatus holds the most recently observed status of the workload identity.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>ObservedGeneration is the most recent generation observed for this workload identity.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
