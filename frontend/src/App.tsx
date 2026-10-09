import {
  useEffect,
  useState,
} from 'react';
import type { FormEvent } from 'react';

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
  Card,
  CardBody,
  CardTitle,
  Label,
} from '@patternfly/react-core';

import '@patternfly/react-core/dist/styles/base.css';

import StreamPlayer from './components/StreamPlayer';

type Stream = {
  id: string;
  name: string;
  url: string;
  state:
  | 'created'
  | 'connecting'
  | 'live'
  | 'stopping'
  | 'stopped'
  | 'error';
  createdAt: string;
};

function App() {
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [streams, setStreams] = useState<Stream[]>([]);

  useEffect(() => {
    const loadStreams = async () => {
      const response = await fetch(
        '/api/v1/streams',
      );

      if (!response.ok) {
        return;
      }

      const existing =
        (await response.json()) as Stream[];

      setStreams(existing);
    };

    void loadStreams();
  }, []);

  const closeAddStream = () => {
    setIsAddOpen(false);
    setName('');
    setUrl('');
  };

  const createStream = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault();

    const response = await fetch(
      '/api/v1/streams',
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          url,
        }),
      },
    );

    if (!response.ok) {
      return;
    }

    const created =
      (await response.json()) as Stream;

    setStreams((current) => [
      ...current,
      created,
    ]);

    closeAddStream();
  };

  const replaceStream = (
    updated: Stream,
  ) => {
    setStreams((current) =>
      current.map((stream) =>
        stream.id === updated.id
          ? updated
          : stream,
      ),
    );
  };

  const refreshStreamUntilSettled = async (
    id: string,
  ) => {
    const maxAttempts = 40;
    const pollIntervalMs = 250;

    for (
      let attempt = 0;
      attempt < maxAttempts;
      attempt += 1
    ) {
      const response = await fetch(
        `/api/v1/streams/${id}`,
      );

      if (!response.ok) {
        return;
      }

      const updated =
        (await response.json()) as Stream;

      replaceStream(updated);

      const isTransitional =
        updated.state === 'connecting' ||
        updated.state === 'stopping';

      if (!isTransitional) {
        return;
      }

      await new Promise<void>((resolve) => {
        setTimeout(
          resolve,
          pollIntervalMs,
        );
      });
    }
  };

  const startStream = async (
    stream: Stream,
  ) => {
    const response = await fetch(
      `/api/v1/streams/${stream.id}/start`,
      {
        method: 'POST',
      },
    );

    if (!response.ok) {
      return;
    }

    await refreshStreamUntilSettled(
      stream.id,
    );
  };

  const stopStream = async (
    stream: Stream,
  ) => {
    const response = await fetch(
      `/api/v1/streams/${stream.id}/stop`,
      {
        method: 'POST',
      },
    );

    if (!response.ok) {
      return;
    }

    await refreshStreamUntilSettled(
      stream.id,
    );
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

      <PageSection>
        {streams.map((stream) => (
          <Card key={stream.id}>
            <CardTitle>
              {stream.name}
            </CardTitle>

            <CardBody>
              {stream.state === 'live' && (
                <StreamPlayer
                  streamId={stream.id}
                  streamName={stream.name}
                  isLive
                />
              )}

              <div>{stream.url}</div>

              <Label>
                {stream.state}
              </Label>

              {(
                stream.state === 'created' ||
                stream.state === 'stopped' ||
                stream.state === 'error'
              ) && (
                  <Button
                    variant="primary"
                    aria-label={`Start ${stream.name}`}
                    onClick={() => {
                      void startStream(stream);
                    }}
                  >
                    Start
                  </Button>
                )}

              {(
                stream.state === 'live' ||
                stream.state === 'connecting'
              ) && (
                  <Button
                    variant="secondary"
                    aria-label={`Stop ${stream.name}`}
                    onClick={() => {
                      void stopStream(stream);
                    }}
                  >
                    Stop
                  </Button>
                )}

              {stream.state === 'stopping' && (
                <Button
                  variant="secondary"
                  isDisabled
                >
                  Stopping
                </Button>
              )}
            </CardBody>
          </Card>
        ))}
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
          <Form
            id="add-stream-form"
            onSubmit={createStream}
          >
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