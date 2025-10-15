import { Server as SocketIOServer } from 'socket.io';
import { Server as HTTPServer } from 'http';
import { KubernetesClient } from './kubernetes-client.js';

export function setupWebSocket(httpServer: HTTPServer) {
  const io = new SocketIOServer(httpServer, {
    cors: {
      origin: '*', // Configure this based on your needs
      methods: ['GET', 'POST']
    },
    path: '/socket.io/'
  });

  io.on('connection', (socket) => {
    console.log(`WebSocket client connected: ${socket.id}`);

    // Watch a specific workflow
    socket.on('watch-workflow', async ({ namespace, name }) => {
      console.log(`Client ${socket.id} watching workflow: ${namespace}/${name}`);
      
      const k8sClient = new KubernetesClient();
      const watchKey = `${namespace}/${name}`;
      
      // Set up periodic polling (Kubernetes watch can be complex, using polling for simplicity)
      const pollInterval = setInterval(async () => {
        try {
          const workflow = await k8sClient.getWorkflow(name, namespace);
          
          const update = {
            workflow: {
              name: workflow.metadata?.name,
              namespace: workflow.metadata?.namespace,
              phase: workflow.status?.phase,
              startTime: workflow.status?.startTime,
              completionTime: workflow.status?.completionTime,
              schedule: workflow.spec?.schedule
            },
            dag: workflow.status?.dag || {
              nodes: [],
              edges: [],
              metadata: {}
            },
            timestamp: new Date().toISOString()
          };
          
          socket.emit('workflow-update', update);
        } catch (error) {
          console.error(`Error watching workflow ${watchKey}:`, error);
          socket.emit('workflow-error', {
            error: error instanceof Error ? error.message : String(error)
          });
        }
      }, 2000); // Poll every 2 seconds

      // Store interval ID for cleanup
      socket.data.watchIntervals = socket.data.watchIntervals || {};
      socket.data.watchIntervals[watchKey] = pollInterval;

      // Send initial data
      try {
        const workflow = await k8sClient.getWorkflow(name, namespace);
        socket.emit('workflow-update', {
          workflow: {
            name: workflow.metadata?.name,
            namespace: workflow.metadata?.namespace,
            phase: workflow.status?.phase,
            startTime: workflow.status?.startTime,
            completionTime: workflow.status?.completionTime
          },
          dag: workflow.status?.dag || { nodes: [], edges: [], metadata: {} },
          timestamp: new Date().toISOString()
        });
      } catch (error) {
        socket.emit('workflow-error', {
          error: error instanceof Error ? error.message : String(error)
        });
      }
    });

    // Stop watching a workflow
    socket.on('unwatch-workflow', ({ namespace, name }) => {
      const watchKey = `${namespace}/${name}`;
      console.log(`Client ${socket.id} unwatching workflow: ${watchKey}`);
      
      if (socket.data.watchIntervals && socket.data.watchIntervals[watchKey]) {
        clearInterval(socket.data.watchIntervals[watchKey]);
        delete socket.data.watchIntervals[watchKey];
      }
    });

    // Handle disconnect
    socket.on('disconnect', () => {
      console.log(`WebSocket client disconnected: ${socket.id}`);
      
      // Clean up all watch intervals
      if (socket.data.watchIntervals) {
        Object.values(socket.data.watchIntervals).forEach((interval: any) => {
          clearInterval(interval);
        });
      }
    });
  });

  console.log('WebSocket server initialized');
  return io;
}








