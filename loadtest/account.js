// Run from the repo root:
//   TARGET=<host>:50051 ACCOUNT_ID=<uuid> VUS=50 DURATION=30s k6 run loadtest/account.js
import grpc from 'k6/net/grpc';
import { check } from 'k6';

const client = new grpc.Client();
client.load(['../protos/proto'], 'account/v1/account.proto');

export const options = {
  scenarios: {
    steady: {
      executor: 'constant-vus',
      vus: Number(__ENV.VUS || 20),
      duration: __ENV.DURATION || '30s',
    },
  },
};

export default () => {
  // One persistent HTTP/2 connection per virtual user, like a real gRPC client.
  if (__ITER === 0) {
    client.connect(__ENV.TARGET, { plaintext: true });
  }
  const res = client.invoke('account.v1.AccountService/GetAccount', { id: __ENV.ACCOUNT_ID });
  check(res, { 'status OK': (r) => r && r.status === grpc.StatusOK });
};
