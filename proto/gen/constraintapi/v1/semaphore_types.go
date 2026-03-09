package constraintapi

// ConstraintApiSemaphoreScope defines the scope for semaphore constraints.
type ConstraintApiSemaphoreScope int32

const (
	ConstraintApiSemaphoreScope_CONSTRAINT_API_SEMAPHORE_SCOPE_UNSPECIFIED ConstraintApiSemaphoreScope = 0
	ConstraintApiSemaphoreScope_CONSTRAINT_API_SEMAPHORE_SCOPE_FUNCTION    ConstraintApiSemaphoreScope = 1
	ConstraintApiSemaphoreScope_CONSTRAINT_API_SEMAPHORE_SCOPE_ENV         ConstraintApiSemaphoreScope = 2
	ConstraintApiSemaphoreScope_CONSTRAINT_API_SEMAPHORE_SCOPE_ACCOUNT     ConstraintApiSemaphoreScope = 3
)

// Enum value maps for ConstraintApiSemaphoreScope.
var (
	ConstraintApiSemaphoreScope_name = map[int32]string{
		0: "CONSTRAINT_API_SEMAPHORE_SCOPE_UNSPECIFIED",
		1: "CONSTRAINT_API_SEMAPHORE_SCOPE_FUNCTION",
		2: "CONSTRAINT_API_SEMAPHORE_SCOPE_ENV",
		3: "CONSTRAINT_API_SEMAPHORE_SCOPE_ACCOUNT",
	}
	ConstraintApiSemaphoreScope_value = map[string]int32{
		"CONSTRAINT_API_SEMAPHORE_SCOPE_UNSPECIFIED": 0,
		"CONSTRAINT_API_SEMAPHORE_SCOPE_FUNCTION":    1,
		"CONSTRAINT_API_SEMAPHORE_SCOPE_ENV":         2,
		"CONSTRAINT_API_SEMAPHORE_SCOPE_ACCOUNT":     3,
	}
)

func (x ConstraintApiSemaphoreScope) Enum() *ConstraintApiSemaphoreScope {
	p := new(ConstraintApiSemaphoreScope)
	*p = x
	return p
}

func (x ConstraintApiSemaphoreScope) String() string {
	if name, ok := ConstraintApiSemaphoreScope_name[int32(x)]; ok {
		return name
	}
	return "UNKNOWN"
}

func (x ConstraintApiSemaphoreScope) Number() int32 {
	return int32(x)
}

// SemaphoreConfig represents the configuration for a semaphore constraint.
type SemaphoreConfig struct {
	Name              string                     `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Scope             ConstraintApiSemaphoreScope `protobuf:"varint,2,opt,name=scope,proto3,enum=constraintapi.v1.ConstraintApiSemaphoreScope" json:"scope,omitempty"`
	KeyExpressionHash string                     `protobuf:"bytes,3,opt,name=key_expression_hash,json=keyExpressionHash,proto3" json:"key_expression_hash,omitempty"`
	Capacity          int32                      `protobuf:"varint,4,opt,name=capacity,proto3" json:"capacity,omitempty"`
}

func (x *SemaphoreConfig) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *SemaphoreConfig) GetScope() ConstraintApiSemaphoreScope {
	if x != nil {
		return x.Scope
	}
	return ConstraintApiSemaphoreScope_CONSTRAINT_API_SEMAPHORE_SCOPE_UNSPECIFIED
}

func (x *SemaphoreConfig) GetKeyExpressionHash() string {
	if x != nil {
		return x.KeyExpressionHash
	}
	return ""
}

func (x *SemaphoreConfig) GetCapacity() int32 {
	if x != nil {
		return x.Capacity
	}
	return 0
}

// SemaphoreConstraint represents an individual semaphore constraint item.
type SemaphoreConstraint struct {
	Name              string                     `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Scope             ConstraintApiSemaphoreScope `protobuf:"varint,2,opt,name=scope,proto3,enum=constraintapi.v1.ConstraintApiSemaphoreScope" json:"scope,omitempty"`
	KeyExpressionHash string                     `protobuf:"bytes,3,opt,name=key_expression_hash,json=keyExpressionHash,proto3" json:"key_expression_hash,omitempty"`
	EvaluatedKeyHash  string                     `protobuf:"bytes,4,opt,name=evaluated_key_hash,json=evaluatedKeyHash,proto3" json:"evaluated_key_hash,omitempty"`
	Amount            int32                      `protobuf:"varint,5,opt,name=amount,proto3" json:"amount,omitempty"`
}

func (x *SemaphoreConstraint) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *SemaphoreConstraint) GetScope() ConstraintApiSemaphoreScope {
	if x != nil {
		return x.Scope
	}
	return ConstraintApiSemaphoreScope_CONSTRAINT_API_SEMAPHORE_SCOPE_UNSPECIFIED
}

func (x *SemaphoreConstraint) GetKeyExpressionHash() string {
	if x != nil {
		return x.KeyExpressionHash
	}
	return ""
}

func (x *SemaphoreConstraint) GetEvaluatedKeyHash() string {
	if x != nil {
		return x.EvaluatedKeyHash
	}
	return ""
}

func (x *SemaphoreConstraint) GetAmount() int32 {
	if x != nil {
		return x.Amount
	}
	return 0
}
