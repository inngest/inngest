// Package descriptor provides functionality to load and parse Protocol Buffer descriptor files.
package descriptor

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"google.golang.org/protobuf/types/descriptorpb"
)

// Loader provides functionality to load field numbers from descriptor files.
type Loader struct {
	descriptorFile string
	files          *protoregistry.Files
}

// NewLoader creates a new descriptor loader for the given descriptor file.
func NewLoader(descriptorFile string) (*Loader, error) {
	if descriptorFile == "" {
		return nil, fmt.Errorf("descriptor file is required")
	}

	data, err := os.ReadFile(descriptorFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read descriptor file %s: %v", descriptorFile, err)
	}

	fileDescSet := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fileDescSet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal descriptor file %s: %v", descriptorFile, err)
	}

	files, err := protodesc.NewFiles(fileDescSet)
	if err != nil {
		return nil, fmt.Errorf("failed to create files from descriptor file %s: %v", descriptorFile, err)
	}

	return &Loader{
		descriptorFile: descriptorFile,
		files:          files,
	}, nil
}

// GetRootMessageDescriptor returns the root message descriptor for the specified messageFullName.
// messageFullName is required and must be a valid full name (e.g., "google.protobuf.Any").
func (l *Loader) GetRootMessageDescriptor(messageFullName string) (protoreflect.MessageDescriptor, error) {
	if l.files == nil {
		return nil, fmt.Errorf("descriptor not loaded, call NewLoader() first")
	}

	if messageFullName == "" {
		// Collect available messages to help user
		var availableMessages []string
		l.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			messages := fd.Messages()
			for i := 0; i < messages.Len(); i++ {
				msg := messages.Get(i)
				availableMessages = append(availableMessages, string(msg.FullName()))
			}
			return true
		})

		if len(availableMessages) == 0 {
			return nil, fmt.Errorf("No messages found in descriptor")
		}
		return nil, fmt.Errorf("message_full_name is required. Available messages: %v", availableMessages)
	}

	// Find specific message type
	desc, err := l.files.FindDescriptorByName(protoreflect.FullName(messageFullName))
	if err != nil {
		return nil, fmt.Errorf("message type %s not found: %v", messageFullName, err)
	}
	if msgDesc, ok := desc.(protoreflect.MessageDescriptor); ok {
		return msgDesc, nil
	}
	return nil, fmt.Errorf("%s is not a message type", messageFullName)
}
