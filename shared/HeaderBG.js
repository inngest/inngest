// This code is taken from FinisherHeader:
// https://www.finisher.co/lab/header/

export const Gradient = (el) => {};

function getOppositeSide(angle, side) {
  const tan = Math.tan(Math.abs(angle) * 0.017453); // pi / 180
  return Math.ceil(side * tan);
}

function hexToRgb(hex) {
  let c;
  if (/^#([A-Fa-f0-9]{3}){1,2}$/.test(hex)) {
    c = hex.substring(1).split("");
    if (c.length === 3) {
      c = [c[0], c[0], c[1], c[1], c[2], c[2]];
    }
    c = "0x" + c.join("");
    return {
      r: (c >> 16) & 255,
      g: (c >> 8) & 255,
      b: c & 255,
    };
  }
  return { r: 0, g: 0, b: 0 };
}

class Particle {
  constructor(hex, quadrant, options) {
    this.o = options;
    this.r = hexToRgb(hex);
    this.d = this.getRandomDir();
    this.h = this.getRandomShape();
    this.s = Math.abs(this.getNrFromRange(this.o.size));
    this.setRandomPositionGivenQuadrant(quadrant);
    this.vx = this.getNrFromRange(this.o.speed.x) * this.getRandomDir();
    this.vy = this.getNrFromRange(this.o.speed.y) * this.getRandomDir();
  }
  setRandomPositionGivenQuadrant(quadrant) {
    const position = this.getRandomPositionInQuadrant();
    if (quadrant === 3) {
      this.x = position.x + position.halfWidth;
      this.y = position.y;
      return;
    }
    if (quadrant === 2) {
      this.x = position.x;
      this.y = position.y + position.halfHeight;
      return;
    }
    if (quadrant === 1) {
      this.x = position.x + position.halfWidth;
      this.y = position.y + position.halfHeight;
      return;
    }
    this.x = position.x;
    this.y = position.y;
  }
  getRandomPositionInQuadrant() {
    const halfWidth = this.o.c.w / 2;
    const halfHeight = this.o.c.h / 2;
    return {
      x: Math.random() * halfWidth,
      y: Math.random() * halfHeight,
      halfHeight,
      halfWidth,
    };
  }
  getNrFromRange(range) {
    if (range.min === range.max) {
      return range.min;
    }
    const diff = range.max - range.min;
    return Math.random() * diff + range.min;
  }
  getRandomDir() {
    return Math.random() > 0.5 ? 1 : -1;
  }
  getRandomShape() {
    return this.o.shapes[Math.floor(Math.random() * this.o.shapes.length)];
  }
  getRgba(rgb, a) {
    return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${a})`;
  }
  animate(ctx, width, height) {
    if (this.o.size.pulse) {
      this.s += this.o.size.pulse * this.d;
      if (this.s > this.o.size.max || this.s < this.o.size.min) {
        this.d *= -1;
      }
      this.s = Math.abs(this.s);
    }
    this.x += this.vx;
    this.y += this.vy;
    if (this.x < 0) {
      this.vx *= -1;
      this.x += 1;
    } else if (this.x > width) {
      this.vx *= -1;
      this.x -= 1;
    }
    if (this.y < 0) {
      this.vy *= -1;
      this.y += 1;
    } else if (this.y > height) {
      this.vy *= -1;
      this.y -= 1;
    }
    ctx.beginPath();
    if (this.o.blending && this.o.blending !== "none") {
      ctx.globalCompositeOperation = this.o.blending;
    }
    const c1 = this.getRgba(this.r, this.o.opacity.center);
    const c2 = this.getRgba(this.r, this.o.opacity.edge);
    const gradientEndRadius =
      this.h === "c"
        ? this.s / 2
        : this.h === "t"
        ? this.s * 0.577
        : this.h === "s"
        ? this.s * 0.707
        : this.s;
    const g = ctx.createRadialGradient(
      this.x,
      this.y,
      0.01,
      this.x,
      this.y,
      gradientEndRadius
    );
    g.addColorStop(0, c1);
    g.addColorStop(1, c2);
    ctx.fillStyle = g;
    const halfSize = Math.abs(this.s / 2);
    if (this.h === "c") {
      ctx.arc(this.x, this.y, halfSize, 0, 6.283185, false); // pi * 2
    }
    if (this.h === "s") {
      const l = this.x - halfSize;
      const r = this.x + halfSize;
      const t = this.y - halfSize;
      const b = this.y + halfSize;
      ctx.moveTo(l, b);
      ctx.lineTo(r, b);
      ctx.lineTo(r, t);
      ctx.lineTo(l, t);
    }
    if (this.h === "t") {
      const baseToCenter = getOppositeSide(30, halfSize);
      const baseY = this.y + baseToCenter;
      ctx.moveTo(this.x - halfSize, baseY);
      ctx.lineTo(this.x + halfSize, baseY);
      ctx.lineTo(this.x, this.y - baseToCenter * 2);
    }
    ctx.closePath();
    ctx.fill();
  }
}

export class FinisherHeader {
  constructor(options, el) {
    this.el = el;
    this.el.style.position = "relative";
    this.c = document.createElement("canvas");
    this.x = this.c.getContext("2d");
    this.suffix = Math.floor(Math.random() * 100000).toString(16);
    this.c.setAttribute("id", this.suffix);

    el.appendChild(this.c);
    let tm;
    window.addEventListener(
      "resize",
      () => {
        clearTimeout(tm);
        tm = setTimeout(this.resize.bind(this), 150);
      },
      false
    );
    this.init(options);
    window.requestAnimationFrame(this.animate.bind(this));
  }

  resize() {
    this.o.c = {
      w: this.el.clientWidth,
      h: this.el.clientHeight,
    };
    this.c.width = this.o.c.w;
    this.c.height = this.o.c.h;
    const offset = getOppositeSide(this.o.skew, this.o.c.w / 2);
    const transform = `skewY(${this.o.skew}deg) translateY(-${offset}px)`;
    this.c.setAttribute(
      "style",
      `position:absolute;top:0;left:0;right:0;bottom:0;-webkit-transform:${transform};transform:${transform};outline: 1px solid transparent;background-color:rgba(${this.bc.r},${this.bc.g},${this.bc.b},1);`
    );
  }

  init(options) {
    this.o = options;
    this.bc = hexToRgb(this.o.colors.background);
    this.particles = [];
    this.resize();
    this.createParticles();
  }

  createParticles() {
    let curColor = 0;
    this.particles = [];
    this.o.ac =
      window.innerWidth < 600 && this.o.count > 5
        ? Math.round(this.o.count / 2)
        : this.o.count;
    for (let i = 0; i < this.o.ac; i++) {
      const quadrant = i % 4;
      const item = new Particle(
        this.o.colors.particles[curColor],
        quadrant,
        this.o
      );
      if (++curColor >= this.o.colors.particles.length) {
        curColor = 0;
      }
      this.particles[i] = item;
    }
  }

  animate() {
    window.requestAnimationFrame(this.animate.bind(this));
    this.x.clearRect(0, 0, this.o.c.w, this.o.c.h);
    for (let i = 0; i < this.o.ac; i++) {
      const item = this.particles[i];
      item.animate(this.x, this.o.c.w, this.o.c.h);
    }
  }
}

globalThis.FinisherHeader = FinisherHeader;
