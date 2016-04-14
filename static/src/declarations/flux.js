declare module 'flux' {
  declare class Dispatcher {
    dispatch(data: any): void;
    register(handler: (payload: any) => void): void;
  }
}
