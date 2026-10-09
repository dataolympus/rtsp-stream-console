import { useState } from 'react';

import {
  Button,
  Form,
  FormGroup,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  Page,
  PageSection,
  TextInput,
  Title,
} from '@patternfly/react-core';

import '@patternfly/react-core/dist/styles/base.css';

function App() {
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');

  const closeAddStream = () => {
    setIsAddOpen(false);
    setName('');
    setUrl('');
  };

  return (
    <Page>
      <PageSection>
        <Title headingLevel="h1">
          RTSP Stream Console
        </Title>
      </PageSection>

      <PageSection>
        <Button
          variant="primary"
          onClick={() => setIsAddOpen(true)}
        >
          Add stream
        </Button>
      </PageSection>

      <Modal
        isOpen={isAddOpen}
        onClose={closeAddStream}
        aria-labelledby="add-stream-title"
      >
        <ModalHeader
          title="Add stream"
          labelId="add-stream-title"
        />

        <ModalBody>
          <Form id="add-stream-form">
            <FormGroup
              label="Name"
              fieldId="stream-name"
              isRequired
            >
              <TextInput
                id="stream-name"
                value={name}
                onChange={(_event, value) => {
                  setName(value);
                }}
                isRequired
              />
            </FormGroup>

            <FormGroup
              label="RTSP URL"
              fieldId="stream-url"
              isRequired
            >
              <TextInput
                id="stream-url"
                type="url"
                value={url}
                onChange={(_event, value) => {
                  setUrl(value);
                }}
                placeholder="rtsp://localhost:8554/camera-1"
                isRequired
              />
            </FormGroup>
          </Form>
        </ModalBody>

        <ModalFooter>
          <Button
            variant="primary"
            form="add-stream-form"
            type="submit"
          >
            Create stream
          </Button>

          <Button
            variant="link"
            onClick={closeAddStream}
          >
            Cancel
          </Button>
        </ModalFooter>
      </Modal>
    </Page>
  );
}

export default App;