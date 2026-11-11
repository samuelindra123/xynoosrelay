export function Signal() {
  return (
    <div className="signal-art" aria-hidden="true">
      <div className="signal-orbit orbit-a" />
      <div className="signal-orbit orbit-b" />
      <div className="signal-orbit orbit-c" />
      <div className="signal-core">
        <span></span>
        <span></span>
      </div>
      <span className="signal-satellite satellite-a">&#10035;</span>
      <span className="signal-satellite satellite-b">&#9825;</span>
      <div className="signal-caption">
        SEDIKIT TEKNOLOGI.
        <br />
        BANYAK RASA.
      </div>
    </div>
  );
}
