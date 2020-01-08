from .. import fileio
import asyncio

_None = object()

class AsyncClient(asyncio.Protocol):
    def __init__(self, *, endian = fileio.FileStream.BIG_ENDIAN, loop = _None):
        super().__init__()

        self.loop              = asyncio.get_event_loop() if loop is _None else loop
        self.transport         = None
        self.recvBuffer        = fileio.FileStream(b'')
        self.recvBuffer.Endian = endian

    async def create_connection(self, address, port):
        return await self.loop.create_connection(lambda : self, address, port)

    def print_exception(self):
        import traceback
        traceback.print_exception(*sys.exc_info())

    ##################################################
    # asyncio.Protocol interface
    ##################################################

    def connection_made(self, transport):
        self.transport = transport

        try:
            self.connectionMade(transport)
        except:
            self.print_exception()
            raise

    def data_received(self, data):
        self.recvBuffer.Position = self.recvBuffer.END_OF_FILE
        self.recvBuffer.Write(data)
        self.recvBuffer.Position = 0

        try:
            self.dataReceived()
        except:
            self.print_exception()
            raise

    def eof_received(self):
        pass

    def connection_lost(self, exc):
        try:
            self.connectionLost(exc)
        except:
            self.print_exception()
            raise

        self.transport.close()
        self.transport = None

    ##################################################
    # asyncio.Protocol interface end
    ##################################################

    def connectionMade(self, transport):
        pass

    def dataReceived(self):
        pass

    def connectionLost(self, exc):
        pass
