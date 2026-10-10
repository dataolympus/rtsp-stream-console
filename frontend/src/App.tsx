import {
  useEffect,
  useState,
} from 'react';
import type { FormEvent } from 'react';

import {
  Alert,
  AlertVariant,
  Button,
  Card,
  CardBody,
  CardTitle,
  EmptyState,
  EmptyStateActions,
  EmptyStateBody,
  EmptyStateFooter,
  Form,
  FormGroup,
  Grid,
  GridItem,
  Label,
  Modal,
  ModalBody,
  ModalHeader,
  ModalFooter,
  Page,
  PageSection,
  TextInput,
  Title,
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
  error?: string;
};

const STREAM_RECONCILE_MS = 2000;

const readAPIError = async (
  response: Response,
): Promise<string | null> => {
  try {
    const payload =
      (await response.json()) as {
        error?: unknown;
      };

    if (
      typeof payload.error === 'string' &&
      payload.error.trim() !== ''
    ) {
      return payload.error.trim();
    }
  } catch {
    // Fall back to the generic action error.
  }

  return null;
};

function App() {
  const [isAddOpen, setIsAddOpen] = useState(false);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [streams, setStreams] = useState<Stream[]>([]);

  const [streamActionErrors, setStreamActionErrors] =
    useState<Record<string, string>>({});

  const clearStreamActionError = (streamId: string) => {
    setStreamActionErrors((current) => {
      const next = { ...current };
      delete next[streamId];

      return next;
    });
  };

  const setStreamActionError = (
    streamId: string,
    message: string,
  ) => {
    setStreamActionErrors((current) => ({
      ...current,
      [streamId]: message,
    }));
  };

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

  const refreshStream = async (
    id: string,
  ): Promise<Stream | null> => {
    const response = await fetch(
      `/api/v1/streams/${id}`,
    );

    if (!response.ok) {
      return null;
    }

    const updated =
      (await response.json()) as Stream;

    replaceStream(updated);

    return updated;
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
      const updated =
        await refreshStream(id);

      if (!updated) {
        return;
      }

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

  useEffect(() => {
    const streamIdsToRefresh = streams
      .filter(
        (stream) =>
          stream.state === 'connecting' ||
          stream.state === 'live' ||
          stream.state === 'stopping',
      )
      .map((stream) => stream.id);

    if (streamIdsToRefresh.length === 0) {
      return;
    }

    const intervalId = window.setInterval(() => {
      for (const streamId of streamIdsToRefresh) {
        void refreshStream(streamId).catch(() => {
          // Preserve the last known state if a
          // background reconciliation request fails.
        });
      }
    }, STREAM_RECONCILE_MS);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [streams]);

  const startStream = async (
    stream: Stream,
  ) => {
    clearStreamActionError(stream.id);

    try {
      const response = await fetch(
        `/api/v1/streams/${stream.id}/start`,
        {
          method: 'POST',
        },
      );

      if (!response.ok) {
        const reason =
          await readAPIError(response);

        setStreamActionError(
          stream.id,
          reason
            ? `Unable to start ${stream.name}: ${reason}`
            : `Unable to start ${stream.name}. Please try again.`,
        );

        return;
      }

      await refreshStreamUntilSettled(
        stream.id,
      );
    } catch {
      setStreamActionError(
        stream.id,
        `Unable to start ${stream.name}. Please try again.`,
      );
    }
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
      <PageSection className="console-header">
        <div className="console-header__content">
          <div>
            <Title headingLevel="h1">
              RTSP Stream Console
            </Title>

            <p className="console-header__description">
              Monitor and control RTSP video streams.
            </p>
          </div>

          <Button
            variant="primary"
            onClick={() => setIsAddOpen(true)}
          >
            Add stream
          </Button>
        </div>
      </PageSection>

      <PageSection className="console-streams">
        {streams.length === 0 ? (
          <EmptyState
            titleText="No streams yet"
            headingLevel="h2"
          >
            <EmptyStateBody>
              Add an RTSP source to begin monitoring video.
            </EmptyStateBody>

            <EmptyStateFooter>
              <EmptyStateActions>
                <Button
                  variant="primary"
                  onClick={() => setIsAddOpen(true)}
                >
                  Add first stream
                </Button>
              </EmptyStateActions>
            </EmptyStateFooter>
          </EmptyState>
        ) : (
          <Grid hasGutter>
            {streams.map((stream) => (
              <GridItem
                key={stream.id}
                span={12}
                lg={6}
                xl={4}
              >
                <Card className="stream-card">
                  <CardTitle>
                    {stream.name}
                  </CardTitle>

                  <CardBody className="stream-card__body">
                    {stream.state === 'live' && (
                      <StreamPlayer
                        streamId={stream.id}
                        streamName={stream.name}
                        isLive
                      />
                    )}

                    <div
                      className="stream-card__url"
                      title={stream.url}
                    >
                      {stream.url}
                    </div>

                    {stream.state === 'error' && stream.error && (
                      <Alert
                        variant={AlertVariant.danger}
                        isInline
                        title={stream.error}
                      />
                    )}

                    {streamActionErrors[stream.id] && (
                      <Alert
                        variant={AlertVariant.danger}
                        isInline
                        title={streamActionErrors[stream.id]}
                      />
                    )}

                    <div className="stream-card__controls">
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
                    </div>
                  </CardBody>
                </Card>
              </GridItem>
            ))}
          </Grid>
        )}
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