import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';

const PROTO_DIR = path.resolve(__dirname, '../../../proto');

const loaderOptions: protoLoader.Options = {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
};

export function loadProto(protoFile: string): grpc.GrpcObject {
  const protoPath = path.join(PROTO_DIR, protoFile);
  const packageDefinition = protoLoader.loadSync(protoPath, loaderOptions);
  return grpc.loadPackageDefinition(packageDefinition);
}

export function createClient<T>(
  proto: grpc.GrpcObject,
  packagePath: string,
  serviceName: string,
  address: string
): T {
  const parts = packagePath.split('.');
  let service: any = proto;
  for (const part of parts) {
    service = service[part];
  }
  const ServiceClient = service[serviceName];
  return new ServiceClient(address, grpc.credentials.createInsecure()) as T;
}
