# k8s-builder-types-gen

A generator for builder functions for kubernetes API types
For testing purposes (but not only) we use builder functions such as

## Features
- Automatic builder generation from `+builder` tags
- Support for ObjectMeta fields (Name, Namespace, Labels, etc.)
- Reduces test code

## Quick Start
```bash
go install github.com/fulviodenza/k8s-builder-types-gen@latest
k8s-builder-types-gen -input-dir=./api/v1 -output-dir=./api/v1
```

## Why This Tool?

Reduces boilerplate when writing unit tests for Kubernetes operators by auto-generating builder functions.

Sometimes, you may want to write functions like these:

```go
// NewPod returns a Pod object with the given options
func NewPod(opts ...func(*Pod)) *Pod {
  obj := &Pod{
    TypeMeta: v1.TypeMeta{
      Kind:       "Pod",
      APIVersion: "v1",
    },
  }

  for _, f := range opts {
    f(obj)
  }

  return obj
}

// WithName sets the name of the Pod
func WithName(name string) func(*Pod) {
  return func(obj *Pod) {
    obj.Name = name
  }
}
```

This helps us to write dynamic objects in tests like the following:

```
pods: []*v1.Pod{
  v1.NewPod(
    v1.WithName("pod-obj"),
  ),
},
```

This tool allows us to not create a static object for each test reducing the cognitive load on the developer, having to navigate through object declaration often in other files or at the end/top of the test file. 

This method allows us to have the whole object declared inside the single testcase and visualizing it since the first sight of it

## Run the generator

Creating the builder functions is straightforward:

```sh
go install github.com/fulviodenza/k8s-builder-types-gen@latest
k8s-builder-types-gen -input-dir=./api/v1 -output-dir=./api/v1
```

This command will generate the builders for all types that are marked with a `// +builder` annotation.

For `.Spec` and `.Status` we will need to have the annotation on the top of the struct and not on the field of the Object:

```go
// +builder
type Pod struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Specification of the desired behavior of the pod.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec PodSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Most recently observed status of the pod.
	// This data may not be up to date.
	// Populated by the system.
	// Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status PodStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +builder
type PodSpec struct {
	// Restart policy for all containers within the pod.
	// One of Always, OnFailure, Never.
	// Default to Always.
	// More info: http://kubernetes.io/docs/user-guide/pod-states#restartpolicy
	// +optional
    // +builder
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty" protobuf:"bytes,3,opt,name=restartPolicy,casttype=RestartPolicy"`
	// Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request.
	// Value must be non-negative integer. The value zero indicates delete immediately.
	// If this value is nil, the default grace period will be used instead.
	// The grace period is the duration in seconds after the processes running in the pod are sent
	// a termination signal and the time when the processes are forcibly halted with a kill signal.
	// Set this value longer than the expected cleanup time for your process.
	// Defaults to 30 seconds.
	// +optional
    // +builder
	TerminationGracePeriodSeconds *int64 `json:"terminationGracePeriodSeconds,omitempty" protobuf:"varint,4,opt,name=terminationGracePeriodSeconds"`
	// Optional duration in seconds the pod may be active on the node relative to
	// StartTime before the system will actively try to mark it failed and kill associated containers.
	// Value must be a positive integer.
	// +optional
    // +builder
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"varint,5,opt,name=activeDeadlineSeconds"`
	// NodeSelector is a selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: http://kubernetes.io/docs/user-guide/node-selection/README
	// +optional
    // +builder
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,7,rep,name=nodeSelector"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this pod.
	// More info: https://kubernetes.io/docs/concepts/security/service-accounts/
	// +optional
    // +builder
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,8,opt,name=serviceAccountName"`
	// DeprecatedServiceAccount is a deprecated alias for ServiceAccountName.
	// Deprecated: Use serviceAccountName instead.
	// +k8s:conversion-gen=false
	// +optional
    // +builder
	DeprecatedServiceAccount string `json:"serviceAccount,omitempty" protobuf:"bytes,9,opt,name=serviceAccount"`

	// NodeName is a request to schedule this pod onto a specific node. If it is non-empty,
	// the scheduler simply schedules this pod onto that node, assuming that it fits resource
	// requirements.
	// +optional
    // +builder
	NodeName string `json:"nodeName,omitempty" protobuf:"bytes,10,opt,name=nodeName"`
	// Host networking requested for this pod. Use the host's network namespace.
	// Default to false.
	// +k8s:conversion-gen=false
	// +optional
    // +builder
	HostNetwork bool `json:"hostNetwork,omitempty" protobuf:"varint,11,opt,name=hostNetwork"`
	// Use the host's pid namespace.
	// Optional: Default to false.
	// +k8s:conversion-gen=false
	// +optional
    // +builder
	HostPID bool `json:"hostPID,omitempty" protobuf:"varint,12,opt,name=hostPID"`
	// Use the host's ipc namespace.
	// Optional: Default to false.
	// +k8s:conversion-gen=false
	// +optional
    // +builder
	HostIPC bool `json:"hostIPC,omitempty" protobuf:"varint,13,opt,name=hostIPC"`
	// Specifies the hostname of the Pod
	// If not specified, the pod's hostname will be set to a system-defined value.
	// +optional
    // +builder
	Hostname string `json:"hostname,omitempty" protobuf:"bytes,16,opt,name=hostname"`
	// If specified, the fully qualified Pod hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>".
	// If not specified, the pod will not have a domainname at all.
	// +optional
    // +builder
	Subdomain string `json:"subdomain,omitempty" protobuf:"bytes,17,opt,name=subdomain"`
	// If specified, the pod will be dispatched by specified scheduler.
	// If not specified, the pod will be dispatched by default scheduler.
	// +optional
    // +builder
	SchedulerName string `json:"schedulername,omitempty" protobuf:"bytes,19,opt,name=schedulername"`
}
```