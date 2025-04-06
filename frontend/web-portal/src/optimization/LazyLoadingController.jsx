import React, { Suspense, lazy } from 'react';

const HeavyComponent = lazy(() => import('./SomeHeavyComponent'));

export default function LazyLoadingController() {
  return (
    <Suspense fallback={<div>Loading...</div>}>
      <HeavyComponent />
    </Suspense>
  );
}
